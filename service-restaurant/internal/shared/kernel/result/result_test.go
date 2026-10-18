package result

import (
	"testing"

	"service-restaurant/internal/shared/kernel/apperror"
)

func TestZeroValueIsInvalid(t *testing.T) {
	var r Result[int]
	if r.IsValid() || r.IsOk() {
		t.Fatal("el valor cero debe ser inválido")
	}
}

func TestOkAndFail(t *testing.T) {
	ok := Ok(7)
	if !ok.IsOk() || ok.Value() != 7 || ok.Err() != nil || !CanCommit(ok) {
		t.Fatal("Ok incorrecto")
	}
	f := Fail[int](apperror.New(apperror.KindConflict, "dup", "duplicado"))
	if !f.IsValid() || f.IsOk() || f.Err().Code != "dup" || CanCommit(f) {
		t.Fatal("Fail incorrecto")
	}
	if Fail[int](nil).IsValid() {
		t.Fatal("Fail(nil) debe ser inválido")
	}
}
