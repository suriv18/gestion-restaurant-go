package pagination

// CursorResult es una página por cursor; NextCursor vacío indica el final.
type CursorResult[T any] struct {
	Items      []T
	NextCursor string
}
