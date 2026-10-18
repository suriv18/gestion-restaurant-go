package security

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"service-restaurant/internal/modules/identidad/application/port/outbound"
)

func TestArgon2HashYVerify(t *testing.T) {
	h, err := NewArgon2Hasher(Argon2ParamsTest)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := h.Hash("Correcta12345")
	if err != nil || !strings.HasPrefix(hash, "$argon2id$v=19$m=8,t=1,p=1$") {
		t.Fatalf("hash=%q err=%v", hash, err)
	}
	if hash2, _ := h.Hash("Correcta12345"); hash2 == hash {
		t.Fatal("la sal debe variar entre hashes")
	}
	if ok, err := h.Verify("Correcta12345", hash); !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if ok, _ := h.Verify("otra", hash); ok {
		t.Fatal("no debe verificar otra contraseña")
	}
	// Los parámetros viajan en el hash: otro hasher con distinto costo lo verifica igual.
	otro, _ := NewArgon2Hasher(Argon2Params{MemoryKiB: 16, Time: 2, Threads: 1, KeyLen: 32, SaltLen: 16})
	if ok, _ := otro.Verify("Correcta12345", hash); !ok {
		t.Fatal("el hash debe ser autocontenido")
	}
	for _, malo := range []string{"", "plano", "$argon2id$v=19$m=8,t=1,p=1$x", "$bcrypt$v=19$m=8,t=1,p=1$YQ$YQ", "$argon2id$v=18$m=8,t=1,p=1$YQ$YQ"} {
		if _, err := h.Verify("x", malo); err == nil {
			t.Errorf("%q debía ser error de formato", malo)
		}
	}
	h.Dummy("lo que sea") // no debe entrar en pánico
}

func jwtCfg() JWTConfig {
	return JWTConfig{Secret: []byte(strings.Repeat("k", 32)), Issuer: "iss", Audience: "aud"}
}

func TestJWTRoundTrip(t *testing.T) {
	j, err := NewJWTIssuer(jwtCfg())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	tok, err := j.Emitir(outbound.TokenClaims{SesionID: 50, UsuarioID: 7}, now, now.Add(12*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	c, ok := j.Verificar(tok, now.Add(time.Minute))
	if !ok || c.SesionID != 50 || c.UsuarioID != 7 {
		t.Fatalf("claims=%+v ok=%v", c, ok)
	}
	if _, ok := j.Verificar(tok, now.Add(12*time.Hour)); ok {
		t.Fatal("vencido en exp")
	}
	if _, ok := j.Verificar(tok, now.Add(-time.Hour)); ok {
		t.Fatal("emitido en el futuro (iat)")
	}
}

func TestJWTRechazaTokensManipulados(t *testing.T) {
	j, _ := NewJWTIssuer(jwtCfg())
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	good, _ := j.Emitir(outbound.TokenClaims{SesionID: 50, UsuarioID: 7}, now, now.Add(time.Hour))

	firmar := func(m jwt.SigningMethod, key any, c jwt.Claims) string {
		s, err := jwt.NewWithClaims(m, c).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	base := func() claims {
		return claims{SesionID: "50", RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "iss", Audience: jwt.ClaimStrings{"aud"}, Subject: "7",
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		}}
	}
	otraClave := []byte(strings.Repeat("x", 32))
	mal := base()
	mal.Issuer = "otro"
	aud := base()
	aud.Audience = jwt.ClaimStrings{"otra"}
	sinExp := base()
	sinExp.ExpiresAt = nil
	sid := base()
	sid.SesionID = "abc"

	casos := map[string]string{
		"firma de otra clave": firmar(jwt.SigningMethodHS256, otraClave, base()),
		"alg none":            firmar(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, base()),
		"alg HS512":           firmar(jwt.SigningMethodHS512, jwtCfg().Secret, base()),
		"emisor distinto":     firmar(jwt.SigningMethodHS256, jwtCfg().Secret, mal),
		"audiencia distinta":  firmar(jwt.SigningMethodHS256, jwtCfg().Secret, aud),
		"sin exp":             firmar(jwt.SigningMethodHS256, jwtCfg().Secret, sinExp),
		"sid no numérico":     firmar(jwt.SigningMethodHS256, jwtCfg().Secret, sid),
		"payload alterado":    good[:len(good)-4] + "AAAA",
		"vacío":               "",
		"basura":              "no.es.jwt",
	}
	for name, tok := range casos {
		if _, ok := j.Verificar(tok, now.Add(time.Minute)); ok {
			t.Errorf("%s: debía rechazarse", name)
		}
	}
}

func TestJWTExigeClaveFuerte(t *testing.T) {
	if _, err := NewJWTIssuer(JWTConfig{Secret: []byte("corta"), Issuer: "i", Audience: "a"}); err == nil {
		t.Fatal("clave corta debe rechazarse")
	}
	if _, err := NewJWTIssuer(JWTConfig{Secret: []byte(strings.Repeat("k", 32))}); err == nil {
		t.Fatal("emisor y audiencia son obligatorios")
	}
}
