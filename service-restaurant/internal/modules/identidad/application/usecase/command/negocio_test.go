package command

import (
	"context"
	"testing"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type negocioStoreFake struct {
	recibido outbound.CambiosNegocio
	llamadas int
}

func (f *negocioStoreFake) Actualizar(_ context.Context, c outbound.CambiosNegocio) (dtoresult.Negocio, bool, error) {
	f.llamadas++
	f.recibido = c
	return dtoresult.Negocio{ID: 1, RUC: "20123456789"}, true, nil
}

func actorNegocio(permiso string) appctx.Actor {
	return appctx.NewActor(7, "ana", 1, []appctx.Grant{{SucursalID: 1, Permiso: permiso}})
}

func texto(s string) *string { return &s }

func TestEditarNegocioRechazos(t *testing.T) {
	admin := actorNegocio(policy.NegocioAdministrar)
	casos := []struct {
		nombre string
		cmd    cmd.EditarNegocio
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", cmd.EditarNegocio{RazonSocial: texto("X")}, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", cmd.EditarNegocio{Actor: actorNegocio(policy.NegocioVer), RazonSocial: texto("X")}, apperror.KindForbidden, "sin_permiso"},
		{"vacío", cmd.EditarNegocio{Actor: admin}, apperror.KindValidation, "sin_cambios"},
		{"razón social en blanco", cmd.EditarNegocio{Actor: admin, RazonSocial: texto("   ")}, apperror.KindValidation, "razon_social_invalida"},
		{"moneda", cmd.EditarNegocio{Actor: admin, Moneda: texto("pe")}, apperror.KindValidation, "moneda_invalida"},
		{"zona horaria", cmd.EditarNegocio{Actor: admin, ZonaHoraria: texto("Marte/Olimpo")}, apperror.KindValidation, "zona_horaria_invalida"},
		{"ubigeo", cmd.EditarNegocio{Actor: admin, Ubigeo: cmd.Campo[string]{Presente: true, Valor: texto("12")}}, apperror.KindValidation, "ubigeo_invalido"},
		{"correo", cmd.EditarNegocio{Actor: admin, Email: cmd.Campo[string]{Presente: true, Valor: texto("no-es-correo")}}, apperror.KindValidation, "email_invalido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			store := &negocioStoreFake{}
			res, err := NewEditarNegocio(store).Execute(context.Background(), c.cmd)
			if err != nil || !res.IsValid() || res.Err() == nil {
				t.Fatalf("debía rechazarse: %v %+v", err, res)
			}
			if res.Err().Kind != c.kind || res.Err().Code != c.code {
				t.Fatalf("rechazo %s/%s, quería %s/%s", res.Err().Kind, res.Err().Code, c.kind, c.code)
			}
			if store.llamadas != 0 {
				t.Fatal("un rechazo no debe tocar el almacenamiento")
			}
		})
	}
}

func TestEditarNegocioNormalizaYVaciaCamposOpcionales(t *testing.T) {
	store := &negocioStoreFake{}
	res, err := NewEditarNegocio(store).Execute(context.Background(), cmd.EditarNegocio{
		Actor:           actorNegocio(policy.NegocioAdministrar),
		NombreComercial: texto("  La Casona  "),
		Moneda:          texto(" pen "),
		Telefono:        cmd.Campo[string]{Presente: true}, // null → vaciar
		Ubigeo:          cmd.Campo[string]{Presente: true, Valor: texto("150101")},
	})
	if err != nil || !res.IsOk() {
		t.Fatalf("debía aceptarse: %v %+v", err, res)
	}
	c := store.recibido
	if *c.NombreComercial != "La Casona" || *c.Moneda != "PEN" {
		t.Errorf("no normalizó: %q %q", *c.NombreComercial, *c.Moneda)
	}
	if !c.Telefono.Presente || c.Telefono.Valor != nil {
		t.Errorf("null debe vaciar el teléfono: %+v", c.Telefono)
	}
	if !c.Ubigeo.Presente || *c.Ubigeo.Valor != "150101" {
		t.Errorf("ubigeo: %+v", c.Ubigeo)
	}
	if c.Email.Presente || c.RazonSocial != nil {
		t.Error("los campos no enviados no deben modificarse")
	}
}
