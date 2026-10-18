// Package pagination contiene DTO de paginación independientes de infraestructura.
package pagination

const (
	DefaultSize = 25
	MaxSize     = 100
)

// PageRequest: page base 0, size entre 1 y MaxSize.
type PageRequest struct {
	Page int
	Size int
}

// Normalize aplica valores por defecto y límites.
func (p PageRequest) Normalize() PageRequest {
	if p.Page < 0 {
		p.Page = 0
	}
	if p.Size <= 0 {
		p.Size = DefaultSize
	}
	if p.Size > MaxSize {
		p.Size = MaxSize
	}
	return p
}

// Offset para SQL paginado.
func (p PageRequest) Offset() int {
	n := p.Normalize()
	return n.Page * n.Size
}
