package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", strings.Repeat("k", 32))
	if _, err := Load(); err == nil {
		t.Fatal("debe exigir DATABASE_URL")
	}
	t.Setenv("DATABASE_URL", "postgres://x")
	c, err := Load()
	if err != nil || c.Addr != ":8080" || c.JWTIssuer != "service-restaurant" {
		t.Fatalf("carga incorrecta: %v %v", c, err)
	}
}

func TestLoadExigeSecretoJWTFuerte(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	for _, s := range []string{"", "corto", strings.Repeat("k", 31)} {
		t.Setenv("JWT_SECRET", s)
		if _, err := Load(); err == nil {
			t.Errorf("JWT_SECRET %q debe rechazarse", s)
		}
	}
}
