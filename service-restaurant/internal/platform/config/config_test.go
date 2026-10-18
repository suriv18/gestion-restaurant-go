package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("debe exigir DATABASE_URL")
	}
	t.Setenv("DATABASE_URL", "postgres://x")
	c, err := Load()
	if err != nil || c.Addr != ":8080" {
		t.Fatalf("carga incorrecta: %v %v", c, err)
	}
}
