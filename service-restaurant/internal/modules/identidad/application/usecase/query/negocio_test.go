package query

import (
	"context"
	"errors"
	"testing"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type negocioReaderFake struct {
	negocio  dtoresult.Negocio
	found    bool
	err      error
	llamadas int
}

func (f *negocioReaderFake) ObtenerNegocio(context.Context) (dtoresult.Negocio, bool, error) {
	f.llamadas++
	return f.negocio, f.found, f.err
}

func actorConPermiso(p string) appctx.Actor {
	return appctx.NewActor(1, "ana", 1, []appctx.Grant{{SucursalID: 1, Permiso: p}})
}

func TestObtenerNegocio(t *testing.T) {
	t.Run("con NEGOCIO_VER", func(t *testing.T) {
		f := &negocioReaderFake{negocio: dtoresult.Negocio{NombreComercial: "Mi negocio"}, found: true}
		res, err := NewObtenerNegocio(f).Execute(context.Background(), qry.ObtenerNegocio{Actor: actorConPermiso(policy.NegocioVer)})
		if err != nil || !res.IsOk() || res.Value().NombreComercial != "Mi negocio" {
			t.Fatalf("debía leerse: %v %+v", err, res.Err())
		}
	})
	t.Run("sin permiso no toca el lector", func(t *testing.T) {
		f := &negocioReaderFake{found: true}
		res, err := NewObtenerNegocio(f).Execute(context.Background(), qry.ObtenerNegocio{Actor: actorConPermiso(policy.SucursalVer)})
		if err != nil || res.Err() == nil || res.Err().Kind != apperror.KindForbidden {
			t.Fatalf("debía ser 403: %v %+v", err, res.Err())
		}
		if f.llamadas != 0 {
			t.Fatalf("el lector se invocó %d veces", f.llamadas)
		}
	})
	t.Run("no encontrado", func(t *testing.T) {
		f := &negocioReaderFake{}
		res, err := NewObtenerNegocio(f).Execute(context.Background(), qry.ObtenerNegocio{Actor: actorConPermiso(policy.NegocioAdministrar)})
		if err != nil || res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
			t.Fatalf("debía ser 404: %v %+v", err, res.Err())
		}
	})
	t.Run("error técnico", func(t *testing.T) {
		f := &negocioReaderFake{err: errors.New("db caída")}
		if _, err := NewObtenerNegocio(f).Execute(context.Background(), qry.ObtenerNegocio{Actor: actorConPermiso(policy.NegocioVer)}); err == nil {
			t.Fatal("debía propagar el error")
		}
	})
}
