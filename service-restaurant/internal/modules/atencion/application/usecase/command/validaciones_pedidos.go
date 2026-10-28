package command

import (
	"strings"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
)

const (
	maxNotasPedido   = 2000
	maxDestinatario  = 150
	maxTelefono      = 30
	maxDireccion     = 500
	maxReferencia    = 250
	maxCanal         = 40
	maxInstrucciones = 2000
)

// textoOpcional valida un texto anulable: nil o solo espacios → ausente; con contenido, se
// recorta y se valida (multilinea admite saltos de línea y tabulaciones).
func textoOpcional(raw *string, max int, multilinea bool, code, msg string) (*string, *apperror.AppError) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	var v string
	var err error
	if multilinea {
		v, err = vo.TextoObligatorio(*raw, max)
	} else {
		v, err = vo.TextoUnaLinea(*raw, max)
	}
	if err != nil {
		return nil, mapper.Validacion(code, msg)
	}
	return &v, nil
}

func validarTipoPedido(raw string) (vo.TipoPedido, *apperror.AppError) {
	t, err := vo.NewTipoPedido(raw)
	if err != nil {
		return "", mapper.Validacion("tipo_invalido", "el tipo debe ser SALON, LLEVAR o DELIVERY")
	}
	return t, nil
}

func validarComensales(raw int) (vo.Comensales, *apperror.AppError) {
	c, err := vo.NewComensales(raw)
	if err != nil {
		return vo.Comensales{}, mapper.Validacion("comensales_invalidos", "los comensales deben estar entre 1 y 500")
	}
	return c, nil
}

func validarNotas(raw *string) (*string, *apperror.AppError) {
	return textoOpcional(raw, maxNotasPedido, true, "notas_invalidas", "las notas admiten hasta 2000 caracteres sin caracteres de control")
}

// validarMesaSegunTipo: SALON exige mesa y los demás tipos no la admiten.
func validarMesaSegunTipo(tipo vo.TipoPedido, mesaID *int64) *apperror.AppError {
	if tipo == vo.TipoSalon && mesaID == nil {
		return mapper.Validacion("mesa_requerida", "el pedido de SALON exige una mesa")
	}
	if tipo != vo.TipoSalon && mesaID != nil {
		return mapper.Validacion("mesa_no_aplica", "solo el pedido de SALON lleva mesa")
	}
	return nil
}

// validarReparto: DELIVERY exige los datos de entrega y los demás tipos no los admiten.
func validarReparto(tipo vo.TipoPedido, r *cmd.Reparto) (*outbound.NuevoReparto, *apperror.AppError) {
	if tipo != vo.TipoDelivery {
		if r != nil {
			return nil, mapper.Validacion("reparto_no_aplica", "solo el pedido DELIVERY lleva datos de reparto")
		}
		return nil, nil
	}
	if r == nil {
		return nil, mapper.Validacion("reparto_requerido", "el pedido DELIVERY exige los datos de reparto")
	}
	dest, err := vo.TextoUnaLinea(r.Destinatario, maxDestinatario)
	if err != nil {
		return nil, mapper.Validacion("destinatario_invalido", "el destinatario es obligatorio (máximo 150 caracteres)")
	}
	tel, err := vo.TextoUnaLinea(r.Telefono, maxTelefono)
	if err != nil {
		return nil, mapper.Validacion("telefono_invalido", "el teléfono es obligatorio (máximo 30 caracteres)")
	}
	dir, err := vo.TextoObligatorio(r.Direccion, maxDireccion)
	if err != nil {
		return nil, mapper.Validacion("direccion_invalida", "la dirección es obligatoria (máximo 500 caracteres)")
	}
	ref, ae := textoOpcional(r.Referencia, maxReferencia, false, "referencia_entrega_invalida", "la referencia admite hasta 250 caracteres en una línea")
	if ae != nil {
		return nil, ae
	}
	canal, ae := textoOpcional(r.Canal, maxCanal, false, "canal_invalido", "el canal admite hasta 40 caracteres en una línea")
	if ae != nil {
		return nil, ae
	}
	return &outbound.NuevoReparto{Destinatario: dest, Telefono: tel, Direccion: dir, Referencia: ref, Canal: canal}, nil
}

func validarCantidad(raw string) (vo.Cantidad, *apperror.AppError) {
	c, err := vo.NewCantidad(raw)
	if err != nil {
		return vo.Cantidad{}, mapper.Validacion("cantidad_invalida", "la cantidad debe ser mayor que cero, con hasta 11 enteros y 3 decimales")
	}
	return c, nil
}

// validarDescuento: nil → 0.
func validarDescuento(raw *string) (vo.Dinero, *apperror.AppError) {
	v := "0"
	if raw != nil {
		v = *raw
	}
	d, err := vo.NewDinero(v)
	if err != nil {
		return vo.Dinero{}, mapper.Validacion("descuento_invalido", "el descuento debe ser un importe con hasta 12 enteros y 2 decimales")
	}
	return d, nil
}

func validarInstrucciones(raw *string) (*string, *apperror.AppError) {
	return textoOpcional(raw, maxInstrucciones, true, "instrucciones_invalidas", "las instrucciones admiten hasta 2000 caracteres sin caracteres de control")
}
