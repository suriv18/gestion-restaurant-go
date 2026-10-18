package security

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"service-restaurant/internal/modules/identidad/application/port/outbound"
)

// MinSecretBytes es el mínimo exigido para la clave HMAC (256 bits).
const MinSecretBytes = 32

// JWTConfig configura la emisión y verificación de tokens.
type JWTConfig struct {
	Secret   []byte
	Issuer   string
	Audience string
}

// JWTIssuer implementa outbound.TokenIssuer con JWT HS256.
//
// El JWT solo identifica la sesión (sid) y al usuario (sub); la vigencia real, la
// revocación y los permisos se comprueban contra la BD en cada petición.
type JWTIssuer struct {
	cfg    JWTConfig
	parser func(now time.Time) *jwt.Parser
}

func NewJWTIssuer(cfg JWTConfig) (*JWTIssuer, error) {
	if len(cfg.Secret) < MinSecretBytes {
		return nil, fmt.Errorf("jwt: la clave debe tener al menos %d bytes", MinSecretBytes)
	}
	if cfg.Issuer == "" || cfg.Audience == "" {
		return nil, errors.New("jwt: emisor y audiencia son obligatorios")
	}
	return &JWTIssuer{
		cfg: cfg,
		parser: func(now time.Time) *jwt.Parser {
			return jwt.NewParser(
				jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), // fija el algoritmo: sin "none" ni RS/HS confundidos
				jwt.WithIssuer(cfg.Issuer),
				jwt.WithAudience(cfg.Audience),
				jwt.WithExpirationRequired(),
				jwt.WithIssuedAt(),
				jwt.WithTimeFunc(func() time.Time { return now }),
			)
		},
	}, nil
}

type claims struct {
	SesionID string `json:"sid"`
	jwt.RegisteredClaims
}

func (j *JWTIssuer) Emitir(c outbound.TokenClaims, ahora, expiraEn time.Time) (string, error) {
	jti := make([]byte, 8)
	if _, err := rand.Read(jti); err != nil {
		return "", fmt.Errorf("jwt: jti: %w", err)
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		SesionID: strconv.FormatInt(c.SesionID, 10),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.cfg.Issuer,
			Audience:  jwt.ClaimStrings{j.cfg.Audience},
			Subject:   strconv.FormatInt(c.UsuarioID, 10),
			IssuedAt:  jwt.NewNumericDate(ahora),
			ExpiresAt: jwt.NewNumericDate(expiraEn),
			ID:        hex.EncodeToString(jti),
		},
	})
	return tok.SignedString(j.cfg.Secret)
}

func (j *JWTIssuer) Verificar(token string, ahora time.Time) (outbound.TokenClaims, bool) {
	var c claims
	t, err := j.parser(ahora).ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return j.cfg.Secret, nil })
	if err != nil || !t.Valid {
		return outbound.TokenClaims{}, false
	}
	sid, err1 := strconv.ParseInt(c.SesionID, 10, 64)
	uid, err2 := strconv.ParseInt(c.Subject, 10, 64)
	if err1 != nil || err2 != nil || sid <= 0 || uid <= 0 {
		return outbound.TokenClaims{}, false
	}
	return outbound.TokenClaims{SesionID: sid, UsuarioID: uid}, true
}
