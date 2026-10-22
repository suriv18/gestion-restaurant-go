package command

import (
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
)

func TestValidarPosicion(t *testing.T) {
	campo := func(v *string) cmd.Campo[string] { return cmd.Campo[string]{Presente: true, Valor: v} }

	if c, ae := validarPosicion("x", cmd.Campo[string]{}); ae != nil || c.Presente {
		t.Fatalf("ausente no se toca: %+v %v", c, ae)
	}
	for _, v := range []*string{nil, texto("")} {
		if c, ae := validarPosicion("x", campo(v)); ae != nil || !c.Presente || c.Valor != nil {
			t.Fatalf("null o vacío vacía el campo: %+v %v", c, ae)
		}
	}
	if c, ae := validarPosicion("y", campo(texto(" -3.5 "))); ae != nil || !c.Presente || *c.Valor != "-3.5" {
		t.Fatalf("con valor se normaliza: %+v %v", c, ae)
	}
	for eje, mayuscula := range map[string]string{"x": "X", "y": "Y"} {
		_, ae := validarPosicion(eje, campo(texto("   ")))
		if ae == nil || ae.Code != "posicion_"+eje+"_invalida" ||
			ae.Message != "la posición "+mayuscula+" debe ser un decimal con hasta 8 enteros y 2 decimales" {
			t.Fatalf("eje %s: %+v", eje, ae)
		}
	}
}
