package repository

import (
	"context"
	"database/sql"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/cargar"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// PedidoSQLReader lee pedidos.
type PedidoSQLReader struct{ db database.DBTX }

func NewPedidoSQLReader(db database.DBTX) *PedidoSQLReader { return &PedidoSQLReader{db} }

// ListarPedidos devuelve las cabeceras de las sedes de f.SedeIDs ordenadas por id (cursor estable).
// Estado y Tipo vacíos no filtran.
func (r *PedidoSQLReader) ListarPedidos(ctx context.Context, f outbound.FiltroPedidos) ([]dtoresult.Pedido, error) {
	var mesa sql.NullInt64
	if f.MesaID != nil {
		mesa = sql.NullInt64{Int64: *f.MesaID, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.Pedido+` FROM sch_restaurante.pedidos
		WHERE sucursal_id = ANY($1::bigint[]) AND id > $2
		  AND ($3::text = '' OR estado = $3) AND ($4::text = '' OR tipo = $4)
		  AND ($5::bigint IS NULL OR mesa_id = $5)
		ORDER BY id LIMIT $6`, f.SedeIDs, f.DespuesDeID, f.Estado, f.Tipo, mesa, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Pedido
	for rows.Next() {
		p, err := columnas.ScanPedido(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PedidoSQLReader) ObtenerPedido(ctx context.Context, id int64) (dtoresult.PedidoDetalle, bool, error) {
	return cargar.Pedido(ctx, r.db, id)
}
