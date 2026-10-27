package command

import (
	"context"
	"errors"
	"strings"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	"service-restaurant/internal/modules/catalogo/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	CodigoProductoDuplicado  = "codigo_duplicado"
	CodigoProductoInactivo   = "producto_inactivo"
	CodigoReferenciaInactiva = "referencia_inactiva"

	maxNombreProducto      = 180
	maxDescripcionProducto = 2000

	mensajeProductoDuplicado   = "ya existe un producto con ese código (aunque esté dado de baja)"
	mensajeDescripcionProducto = "la descripción admite hasta 2000 caracteres"
	mensajeImagenURL           = "imagen_url debe ser una URL http o https de hasta 2000 caracteres"
)

// Texto de cada referencia que el trigger de referencias activas puede rechazar (campo -> texto).
var referenciasProducto = []struct{ campo, texto string }{
	{"categoria_id", "la categoría"},
	{"articulo_id", "el artículo de inventario"},
	{"impuesto_id", "el impuesto"},
}

// deProducto traduce los errores de almacenamiento propios de los productos. El trigger
// a1_referencias_activas rechaza una categoría, un artículo o un impuesto inexistente o dado de
// baja con un mensaje técnico ("Referencia categoria_id=5: maestro categorias ..."): aquí se
// sustituye por uno legible. Al crear o editar es 422 referencia_invalida (el dato enviado es
// inválido); al restaurar es 409 referencia_inactiva (el estado actual de otro maestro lo impide).
func deProducto(err error, contexto string, restaurando bool) (appresult.Result[dtoresult.Producto], error) {
	var regla *outbound.ErrReglaDatos
	switch {
	case errors.Is(err, outbound.ErrProductoDuplicado):
		return rechazar[dtoresult.Producto](apperror.New(apperror.KindConflict, CodigoProductoDuplicado, mensajeProductoDuplicado))
	case errors.As(err, &regla):
		for _, r := range referenciasProducto {
			if !strings.HasPrefix(regla.Mensaje, "Referencia "+r.campo+"=") {
				continue
			}
			if restaurando {
				return rechazar[dtoresult.Producto](apperror.New(apperror.KindConflict, CodigoReferenciaInactiva,
					"no se puede restaurar el producto: "+r.texto+" del producto está dado de baja; restáurelo o cambie la referencia antes"))
			}
			return rechazar[dtoresult.Producto](apperror.New(apperror.KindValidation, mapper.CodigoReferenciaInvalida,
				r.texto+" indicado no existe o está dado de baja"))
		}
	}
	return deAlmacen[dtoresult.Producto](contexto, "", err)
}

func validarCodigoProducto(raw string) (string, *apperror.AppError) {
	c, err := vo.NewCodigoProducto(raw)
	if err != nil {
		return "", mapper.Validacion("codigo_invalido",
			"el código admite letras (se pasa a mayúsculas), dígitos, - _ . y no lleva espacios (1 a 40 caracteres)")
	}
	return c, nil
}

func validarTipoProducto(raw string) (string, *apperror.AppError) {
	t, err := vo.NewTipoProducto(raw)
	if err != nil {
		return "", mapper.Validacion("tipo_invalido", "el tipo debe ser PREPARADO, REVENTA, ADICIONAL o SERVICIO")
	}
	return t, nil
}

func validarPrecio(raw string) (string, *apperror.AppError) {
	p, err := vo.NewPrecioVenta(raw)
	if err != nil {
		return "", mapper.Validacion("precio_invalido",
			"precio_venta debe ser un decimal en cadena de 0 a 99999999999.99 con hasta 2 decimales (sin signo ni exponente)")
	}
	return p, nil
}

func validarFactor(raw string) (string, *apperror.AppError) {
	f, err := vo.NewFactorConsumo(raw)
	if err != nil {
		return "", mapper.Validacion("factor_consumo_invalido",
			"factor_consumo debe ser un decimal en cadena mayor que 0 con hasta 6 decimales (sin signo ni exponente)")
	}
	return f, nil
}

func validarAfectacion(raw string) (string, *apperror.AppError) {
	a, err := vo.NewAfectacionIGV(raw)
	if err != nil {
		return "", mapper.Validacion("afectacion_invalida", "la afectación del IGV debe ser \"10\" (gravado), \"20\" (exonerado) o \"30\" (inafecto)")
	}
	return a, nil
}

var (
	fmtDescripcionProducto         = texto(maxDescripcionProducto, true)
	fmtImagenURL           formato = vo.NewImagenURL
)

// textoOpcionalProducto normaliza un campo anulable: nil o en blanco -> nil; con valor, lo valida.
func textoOpcionalProducto(raw *string, codigo, mensaje string, f formato) (*string, *apperror.AppError) {
	if vacio(raw) {
		return nil, nil
	}
	v, err := f(*raw)
	if err != nil {
		return nil, mapper.Validacion(codigo, mensaje)
	}
	return &v, nil
}

func campoTextoProducto(c cmd.Campo[string], codigo, mensaje string, f formato) (outbound.CampoOpcional, *apperror.AppError) {
	if !c.Presente {
		return outbound.CampoOpcional{}, nil
	}
	v, ae := textoOpcionalProducto(c.Valor, codigo, mensaje, f)
	if ae != nil {
		return outbound.CampoOpcional{}, ae
	}
	return outbound.CampoOpcional{Presente: true, Valor: v}, nil
}

// validarCruces comprueba las invariantes entre campos sobre el valor resultante: REVENTA si y
// solo si hay artículo, y afectación gravada ("10") exige impuesto.
func validarCruces(tipo string, articulo *int64, afectacion string, impuesto *int64) *apperror.AppError {
	switch {
	case tipo == vo.TipoReventa && articulo == nil:
		return mapper.Validacion("articulo_requerido", "un producto REVENTA exige articulo_id")
	case tipo != vo.TipoReventa && articulo != nil:
		return mapper.Validacion("articulo_no_permitido", "solo un producto REVENTA lleva articulo_id")
	case afectacion == vo.AfectacionGravado && impuesto == nil:
		return mapper.Validacion("impuesto_requerido", "la afectación gravada (10) exige impuesto_id")
	}
	return nil
}

// CrearProducto da de alta un producto de la carta.
type CrearProducto struct{ productos outbound.ProductoStore }

func NewCrearProducto(p outbound.ProductoStore) *CrearProducto { return &CrearProducto{p} }

func (uc *CrearProducto) Execute(ctx context.Context, c cmd.CrearProducto) (appresult.Result[dtoresult.Producto], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCarta, policy.CartaAdministrar); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	if c.CategoriaID <= 0 {
		return rechazar[dtoresult.Producto](mapper.Validacion("categoria_requerida", "categoria_id es obligatorio"))
	}
	n := outbound.NuevoProducto{CategoriaID: c.CategoriaID, ArticuloID: c.ArticuloID, ImpuestoID: c.ImpuestoID}
	var ae *apperror.AppError
	if n.Codigo, ae = validarCodigoProducto(c.Codigo); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	if n.Nombre, ae = validarNombreHasta(c.Nombre, "nombre", maxNombreProducto); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	if n.Descripcion, ae = textoOpcionalProducto(c.Descripcion, "descripcion_invalido", mensajeDescripcionProducto, fmtDescripcionProducto); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	if n.Tipo, ae = validarTipoProducto(c.Tipo); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	factor := "1"
	if c.Factor != nil {
		factor = *c.Factor
	}
	if n.Factor, ae = validarFactor(factor); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	if n.Precio, ae = validarPrecio(c.Precio); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	if n.Afectacion, ae = validarAfectacion(c.Afectacion); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	if n.ImagenURL, ae = textoOpcionalProducto(c.ImagenURL, "imagen_url_invalido", mensajeImagenURL, fmtImagenURL); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	if ae = validarCruces(n.Tipo, n.ArticuloID, n.Afectacion, n.ImpuestoID); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	p, err := uc.productos.Crear(ctx, n)
	if err != nil {
		return deProducto(err, "crear producto", false)
	}
	return appresult.Ok(p), nil
}

// EditarProducto modifica solo los campos enviados de un producto activo. Las invariantes
// cruzadas se validan sobre el valor resultante, con la fila leída FOR UPDATE.
type EditarProducto struct{ productos outbound.ProductoStore }

func NewEditarProducto(p outbound.ProductoStore) *EditarProducto { return &EditarProducto{p} }

func (uc *EditarProducto) Execute(ctx context.Context, c cmd.EditarProducto) (appresult.Result[dtoresult.Producto], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCarta, policy.CartaAdministrar); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	actual, found, err := uc.productos.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Producto]("buscar producto", err)
	}
	if !found {
		return rechazar[dtoresult.Producto](mapper.NoEncontrado("producto"))
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Producto](apperror.New(apperror.KindConflict, CodigoProductoInactivo,
			"el producto está dado de baja: restáurelo para editarlo"))
	}
	if c.CategoriaID == nil && c.Codigo == nil && c.Nombre == nil && !c.Descripcion.Presente && c.Tipo == nil &&
		!c.ArticuloID.Presente && c.Factor == nil && c.Precio == nil && c.Afectacion == nil &&
		!c.ImpuestoID.Presente && !c.ImagenURL.Presente {
		return rechazar[dtoresult.Producto](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}

	var cambios outbound.CambiosProducto
	var ae *apperror.AppError
	if c.CategoriaID != nil {
		if *c.CategoriaID <= 0 {
			return rechazar[dtoresult.Producto](mapper.Validacion("categoria_requerida", "categoria_id no es válido"))
		}
		cambios.CategoriaID = c.CategoriaID
	}
	if c.Codigo != nil {
		v, ae := validarCodigoProducto(*c.Codigo)
		if ae != nil {
			return rechazar[dtoresult.Producto](ae)
		}
		cambios.Codigo = &v
	}
	if c.Nombre != nil {
		v, ae := validarNombreHasta(*c.Nombre, "nombre", maxNombreProducto)
		if ae != nil {
			return rechazar[dtoresult.Producto](ae)
		}
		cambios.Nombre = &v
	}
	if cambios.Descripcion, ae = campoTextoProducto(c.Descripcion, "descripcion_invalido", mensajeDescripcionProducto, fmtDescripcionProducto); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	tipo := actual.Tipo
	if c.Tipo != nil {
		if tipo, ae = validarTipoProducto(*c.Tipo); ae != nil {
			return rechazar[dtoresult.Producto](ae)
		}
		cambios.Tipo = &tipo
	}
	if c.Factor != nil {
		v, ae := validarFactor(*c.Factor)
		if ae != nil {
			return rechazar[dtoresult.Producto](ae)
		}
		cambios.Factor = &v
	}
	if c.Precio != nil {
		v, ae := validarPrecio(*c.Precio)
		if ae != nil {
			return rechazar[dtoresult.Producto](ae)
		}
		cambios.Precio = &v
	}
	afectacion := actual.AfectacionIGV
	if c.Afectacion != nil {
		if afectacion, ae = validarAfectacion(*c.Afectacion); ae != nil {
			return rechazar[dtoresult.Producto](ae)
		}
		cambios.Afectacion = &afectacion
	}
	if cambios.ImagenURL, ae = campoTextoProducto(c.ImagenURL, "imagen_url_invalido", mensajeImagenURL, fmtImagenURL); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}

	// Valor resultante de las referencias anulables.
	articulo, impuesto := actual.ArticuloID, actual.ImpuestoID
	if c.ArticuloID.Presente {
		articulo = c.ArticuloID.Valor
		cambios.ArticuloID = outbound.CampoID{Presente: true, Valor: articulo}
	}
	if c.ImpuestoID.Presente {
		impuesto = c.ImpuestoID.Valor
		cambios.ImpuestoID = outbound.CampoID{Presente: true, Valor: impuesto}
	}
	if ae = validarCruces(tipo, articulo, afectacion, impuesto); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}

	p, found, err := uc.productos.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deProducto(err, "editar producto", false)
	}
	if !found {
		return rechazar[dtoresult.Producto](mapper.NoEncontrado("producto"))
	}
	return appresult.Ok(p), nil
}

// CambiarEstadoProducto da de baja (con motivo, RES-RF-030) o restaura un producto. La
// restauración conserva el mismo id; el trigger de referencias activas la rechaza si la
// categoría, el artículo o el impuesto del producto siguen dados de baja.
type CambiarEstadoProducto struct{ productos outbound.ProductoStore }

func NewCambiarEstadoProducto(p outbound.ProductoStore) *CambiarEstadoProducto {
	return &CambiarEstadoProducto{p}
}

func (uc *CambiarEstadoProducto) Execute(ctx context.Context, c cmd.CambiarEstadoProducto) (appresult.Result[dtoresult.Producto], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCarta, policy.CartaAdministrar); ae != nil {
		return rechazar[dtoresult.Producto](ae)
	}
	var motivo string
	if !c.Activo {
		m, ae := validarMotivo(c.Motivo)
		if ae != nil {
			return rechazar[dtoresult.Producto](ae)
		}
		motivo = m
	}
	actual, found, err := uc.productos.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Producto]("buscar producto", err)
	}
	if !found {
		return rechazar[dtoresult.Producto](mapper.NoEncontrado("producto"))
	}
	var p dtoresult.Producto
	switch {
	case c.Activo && actual.EsActivo:
		return rechazar[dtoresult.Producto](apperror.New(apperror.KindConflict, CodigoYaActiva, "el producto ya está activo"))
	case !c.Activo && !actual.EsActivo:
		return rechazar[dtoresult.Producto](apperror.New(apperror.KindConflict, CodigoYaDadaDeBaja, "el producto ya está dado de baja"))
	case c.Activo:
		p, found, err = uc.productos.Restaurar(ctx, c.ID)
	default:
		p, found, err = uc.productos.DarBaja(ctx, c.ID, motivo)
	}
	if err != nil {
		return deProducto(err, "cambiar estado del producto", c.Activo)
	}
	if !found {
		return rechazar[dtoresult.Producto](mapper.NoEncontrado("producto"))
	}
	return appresult.Ok(p), nil
}
