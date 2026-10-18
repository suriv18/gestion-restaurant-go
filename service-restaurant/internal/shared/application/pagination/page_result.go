package pagination

// PageResult es una página con total.
type PageResult[T any] struct {
	Items []T
	Page  int
	Size  int
	Total int64
}
