package command

import (
	"strings"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/vo"
)

// opcionalValidado normaliza un campo anulable de un PATCH: ausente → no se toca; null o en
// blanco → se vacía; con valor → se valida con norm. ok=false si el valor es inválido.
func opcionalValidado(c cmd.Campo[string], norm func(string) (string, bool)) (outbound.CampoOpcional, bool) {
	if !c.Presente {
		return outbound.CampoOpcional{}, true
	}
	if c.Valor == nil || strings.TrimSpace(*c.Valor) == "" {
		return outbound.CampoOpcional{Presente: true}, true
	}
	v, ok := norm(*c.Valor)
	if !ok {
		return outbound.CampoOpcional{}, false
	}
	return outbound.CampoOpcional{Presente: true, Valor: &v}, true
}

// valorOpcional valida un texto anulable de un alta: nil o en blanco → nil.
func valorOpcional(p *string, norm func(string) (string, bool)) (*string, bool) {
	c, ok := opcionalValidado(cmd.Campo[string]{Presente: true, Valor: p}, norm)
	return c.Valor, ok
}

func normUbigeo(s string) (string, bool) {
	u, err := vo.NewUbigeo(s)
	return u.String(), err == nil
}

func normMax(max int) func(string) (string, bool) {
	return func(s string) (string, bool) {
		v, err := vo.TextoObligatorio(s, max)
		return v, err == nil
	}
}

func normEmail(s string) (string, bool) {
	v := strings.TrimSpace(s)
	e, ok := emailOpcional(&v)
	if !ok || e == nil {
		return "", false
	}
	return *e, true
}
