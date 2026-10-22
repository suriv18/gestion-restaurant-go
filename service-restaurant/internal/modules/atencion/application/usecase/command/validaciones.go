package command

import (
	"strings"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
)

// Validaciones de campos compartidas por el alta y la edición: cada una devuelve el valor ya
// normalizado o el rechazo de validación (422) con el código y el mensaje del campo.

func validarNombreZona(raw string) (string, *apperror.AppError) {
	n, err := vo.TextoUnaLinea(raw, maxNombreZona)
	if err != nil {
		return "", mapper.Validacion("nombre_invalido", "el nombre es obligatorio (máximo 80 caracteres)")
	}
	return n, nil
}

func validarOrden(raw int) (int, *apperror.AppError) {
	o, err := vo.NewOrden(raw)
	if err != nil {
		return 0, mapper.Validacion("orden_invalido", "el orden de visualización debe estar entre 0 y 9999")
	}
	return o.Int(), nil
}

func validarNumeroMesa(raw string) (string, *apperror.AppError) {
	n, err := vo.NewNumeroMesa(raw)
	if err != nil {
		return "", mapper.Validacion("numero_mesa_invalido", "el número de mesa es obligatorio (máximo 20 caracteres)")
	}
	return n.String(), nil
}

func validarCapacidad(raw int) (int, *apperror.AppError) {
	c, err := vo.NewCapacidad(raw)
	if err != nil {
		return 0, mapper.Validacion("capacidad_invalida", "la capacidad debe estar entre 1 y 500")
	}
	return c.Int(), nil
}

// validarPosicion valida la posición de un eje ("x" o "y") anulable: ausente → no se toca; null
// o cadena vacía → se vacía; con valor → se valida y se normaliza (un texto solo de espacios no
// es un decimal: se rechaza).
func validarPosicion(eje string, c cmd.Campo[string]) (outbound.CampoOpcional, *apperror.AppError) {
	if !c.Presente {
		return outbound.CampoOpcional{}, nil
	}
	if c.Valor == nil || *c.Valor == "" {
		return outbound.CampoOpcional{Presente: true}, nil
	}
	p, err := vo.NewPosicion(*c.Valor)
	if err != nil {
		return outbound.CampoOpcional{}, mapper.Validacion("posicion_"+eje+"_invalida",
			"la posición "+strings.ToUpper(eje)+" debe ser un decimal con hasta 8 enteros y 2 decimales")
	}
	v := p.String()
	return outbound.CampoOpcional{Presente: true, Valor: &v}, nil
}
