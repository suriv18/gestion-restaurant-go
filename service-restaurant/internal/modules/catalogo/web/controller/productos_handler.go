package controller

import (
	"fmt"
	"net/http"
	"net/url"

	"service-restaurant/internal/modules/catalogo/application/dto/command"
	"service-restaurant/internal/modules/catalogo/application/dto/query"
	"service-restaurant/internal/modules/catalogo/application/port/inbound"
	"service-restaurant/internal/modules/catalogo/web/dto/request"
	webmapper "service-restaurant/internal/modules/catalogo/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
	"service-restaurant/internal/shared/kernel/apperror"
)

// ProductosHandler: productos de la carta, maestro global (RES-RF-005, RES-RF-030).
type ProductosHandler struct {
	listar   inbound.ListarProductos
	obtener  inbound.ObtenerProducto
	crear    inbound.CrearProducto
	editar   inbound.EditarProducto
	estado   inbound.CambiarEstadoProducto
	disponib inbound.CambiarDisponibilidad
}

func NewProductosHandler(l inbound.ListarProductos, o inbound.ObtenerProducto, c inbound.CrearProducto,
	e inbound.EditarProducto, s inbound.CambiarEstadoProducto, d inbound.CambiarDisponibilidad) *ProductosHandler {
	return &ProductosHandler{l, o, c, e, s, d}
}

// idOpcional convierte un id opcional en cadena; nil se queda en nil.
func idOpcional(valor *string, campo string) (*int64, error) {
	if valor == nil {
		return nil, nil
	}
	n, err := idCadena(*valor, campo)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// campoID convierte un campo anulable de PATCH (ausente / null / id) al comando.
func campoID(o request.Opcional[string], campo string) (command.Campo[int64], error) {
	if !o.Presente {
		return command.Campo[int64]{}, nil
	}
	v, err := idOpcional(o.Valor, campo)
	if err != nil {
		return command.Campo[int64]{}, err
	}
	return command.Campo[int64]{Presente: true, Valor: v}, nil
}

// activoProductos lee es_activo: "true", "false" o la extensión "todos" (sin filtro de estado).
// Sin parámetro el listado devuelve solo los activos (spec D6).
func activoProductos(valores map[string][]string) (activo *bool, todos bool, err error) {
	if vs := valores["es_activo"]; len(vs) > 0 && vs[0] == "todos" {
		return nil, true, nil
	}
	activo, err = activoDe(valores)
	return activo, false, err
}

// sedeOpcional lee sucursal_id de la consulta (ausente = nil); mal formado es 400.
func sedeOpcional(q url.Values) (*int64, error) {
	v := q.Get("sucursal_id")
	if v == "" {
		return nil, nil
	}
	n, err := idCadena(v, "sucursal_id")
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// disponibleDe lee el filtro opcional disponible ("true" o "false").
func disponibleDe(q url.Values) (*bool, error) {
	switch q.Get("disponible") {
	case "":
		return nil, nil
	case "true":
		t := true
		return &t, nil
	case "false":
		f := false
		return &f, nil
	}
	return nil, apperror.New(apperror.KindBadRequest, "disponible_invalido", "disponible debe ser true o false")
}

// Listar: GET /productos?q=&categoria_id=&es_activo=&sucursal_id=&disponible=&limit=&cursor=
func (h *ProductosHandler) Listar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	q, err := consulta(r)
	if err != nil {
		return err
	}
	limite, err := limiteDe(q)
	if err != nil {
		return err
	}
	activo, todos, err := activoProductos(q)
	if err != nil {
		return err
	}
	var categoria *int64
	if v := q.Get("categoria_id"); v != "" {
		n, err := idCadena(v, "categoria_id")
		if err != nil {
			return apperror.New(apperror.KindBadRequest, "categoria_id_invalido", "categoria_id debe ser un id numérico")
		}
		categoria = &n
	}
	sede, err := sedeOpcional(q)
	if err != nil {
		return err
	}
	disponible, err := disponibleDe(q)
	if err != nil {
		return err
	}
	res, err := h.listar.Execute(r.Context(), query.ListarProductos{
		Actor: actor, Activo: activo, Todos: todos, CategoriaID: categoria, Texto: q.Get("q"), SucursalID: sede, Disponible: disponible, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToProductosPagina)
}

// Obtener: GET /productos/{id}?sucursal_id=
func (h *ProductosHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	q, err := consulta(r)
	if err != nil {
		return err
	}
	sede, err := sedeOpcional(q)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerProducto{Actor: actor, ID: id, SucursalID: sede})
	return responder(w, res, err, http.StatusOK, webmapper.ToProducto)
}

// Crear: POST /productos (Idempotency-Key obligatoria)
func (h *ProductosHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.ProductoAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	// categoria_id ausente llega vacío al caso de uso (422 categoria_requerida); mal formado es 400.
	var categoria int64
	if body.CategoriaID != "" {
		if categoria, err = idCadena(body.CategoriaID, "categoria_id"); err != nil {
			return err
		}
	}
	articulo, err := idOpcional(body.ArticuloID, "articulo_id")
	if err != nil {
		return err
	}
	impuesto, err := idOpcional(body.ImpuestoID, "impuesto_id")
	if err != nil {
		return err
	}
	in, err := intencion(r, "POST /productos", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearProducto{
		Intencion: in, Actor: actor, CategoriaID: categoria, Codigo: body.Codigo, Nombre: body.Nombre,
		Descripcion: body.Descripcion, Tipo: body.Tipo, ArticuloID: articulo, Factor: body.FactorConsumo,
		Precio: body.PrecioVenta, Afectacion: body.AfectacionIGV, ImpuestoID: impuesto, ImagenURL: body.ImagenURL})
	return responder(w, res, err, http.StatusCreated, webmapper.ToProducto)
}

// Editar: PATCH /productos/{id}
func (h *ProductosHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.ProductoEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	categoria, err := idOpcional(body.CategoriaID, "categoria_id")
	if err != nil {
		return err
	}
	articulo, err := campoID(body.ArticuloID, "articulo_id")
	if err != nil {
		return err
	}
	impuesto, err := campoID(body.ImpuestoID, "impuesto_id")
	if err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarProducto{
		Actor: actor, ID: id, CategoriaID: categoria, Codigo: body.Codigo, Nombre: body.Nombre,
		Descripcion: campo(body.Descripcion.Presente, body.Descripcion.Valor), Tipo: body.Tipo, ArticuloID: articulo,
		Factor: body.FactorConsumo, Precio: body.PrecioVenta, Afectacion: body.AfectacionIGV, ImpuestoID: impuesto,
		ImagenURL: campo(body.ImagenURL.Presente, body.ImagenURL.Valor)})
	return responder(w, res, err, http.StatusOK, webmapper.ToProducto)
}

// Baja: POST /productos/{id}/baja con {"motivo": "..."} (Idempotency-Key obligatoria)
func (h *ProductosHandler) Baja(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.Motivo
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, fmt.Sprintf("POST /productos/%d/baja", id), body)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoProducto{
		Intencion: in, Actor: actor, ID: id, Activo: false, Motivo: body.Motivo})
	return responder(w, res, err, http.StatusOK, webmapper.ToProducto)
}

// Restauracion: POST /productos/{id}/restauracion (Idempotency-Key obligatoria; cuerpo vacío o "{}")
func (h *ProductosHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var c request.Confirmacion
	if err := leerCuerpoOpcionalVacio(r, &c); err != nil {
		return err
	}
	in, err := intencion(r, fmt.Sprintf("POST /productos/%d/restauracion", id), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoProducto{Intencion: in, Actor: actor, ID: id, Activo: true})
	return responder(w, res, err, http.StatusOK, webmapper.ToProducto)
}

// Disponibilidad: PUT /productos/{id}/disponibilidad. Idempotente por naturaleza: no lleva
// Idempotency-Key. Hace upsert por (producto, sucursal) y devuelve siempre la fila.
func (h *ProductosHandler) Disponibilidad(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.Disponibilidad
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	// sucursal_id ausente llega como 0 al caso de uso (422 sucursal_requerida); mal formado es 400.
	var sede int64
	if body.SucursalID != "" {
		if sede, err = idCadena(body.SucursalID, "sucursal_id"); err != nil {
			return err
		}
	}
	estacion, err := campoID(body.EstacionID, "estacion_id")
	if err != nil {
		return err
	}
	res, err := h.disponib.Execute(r.Context(), command.CambiarDisponibilidad{
		Actor: actor, ProductoID: id, SucursalID: sede, Disponible: body.Disponible,
		Precio: campo(body.PrecioVenta.Presente, body.PrecioVenta.Valor), Estacion: estacion})
	return responder(w, res, err, http.StatusOK, webmapper.ToProductoSucursal)
}
