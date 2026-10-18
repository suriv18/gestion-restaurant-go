// Package security implementa hash de contraseñas (argon2id), JWT y reloj.
package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params fija el costo. Los hashes guardan sus propios parámetros (PHC), por lo
// que subirlos más adelante no invalida contraseñas existentes.
type Argon2Params struct {
	MemoryKiB uint32
	Time      uint32
	Threads   uint8
	KeyLen    uint32
	SaltLen   uint32
}

// Argon2ParamsDefault: 64 MiB, 3 pasadas, 2 hilos.
var Argon2ParamsDefault = Argon2Params{MemoryKiB: 64 * 1024, Time: 3, Threads: 2, KeyLen: 32, SaltLen: 16}

// Argon2ParamsTest es un costo mínimo solo para pruebas.
var Argon2ParamsTest = Argon2Params{MemoryKiB: 8, Time: 1, Threads: 1, KeyLen: 32, SaltLen: 16}

var b64 = base64.RawStdEncoding

// Argon2Hasher implementa outbound.PasswordHasher.
type Argon2Hasher struct {
	p     Argon2Params
	dummy string
}

func NewArgon2Hasher(p Argon2Params) (*Argon2Hasher, error) {
	h := &Argon2Hasher{p: p}
	d, err := h.Hash("dummy-password-nunca-valida")
	if err != nil {
		return nil, err
	}
	h.dummy = d
	return h, nil
}

// Hash devuelve el hash en formato PHC: $argon2id$v=19$m=..,t=..,p=..$salt$hash
func (h *Argon2Hasher) Hash(plain string) (string, error) {
	salt := make([]byte, h.p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("argon2: salt: %w", err)
	}
	key := argon2.IDKey([]byte(plain), salt, h.p.Time, h.p.MemoryKiB, h.p.Threads, h.p.KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.p.MemoryKiB, h.p.Time, h.p.Threads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// Verify compara en tiempo constante. Un hash mal formado es un error técnico.
func (h *Argon2Hasher) Verify(plain, encoded string) (bool, error) {
	p, salt, want, err := decode(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(plain), salt, p.Time, p.MemoryKiB, p.Threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// Dummy gasta el costo de una verificación real.
func (h *Argon2Hasher) Dummy(plain string) { _, _ = h.Verify(plain, h.dummy) }

var errFormato = errors.New("argon2: hash con formato no reconocido")

func decode(encoded string) (Argon2Params, []byte, []byte, error) {
	var p Argon2Params
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return p, nil, nil, errFormato
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return p, nil, nil, errFormato
	}
	var threads uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Time, &threads); err != nil || threads == 0 || threads > 255 {
		return p, nil, nil, errFormato
	}
	p.Threads = uint8(threads)
	salt, err := b64.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return p, nil, nil, errFormato
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return p, nil, nil, errFormato
	}
	return p, salt, key, nil
}
