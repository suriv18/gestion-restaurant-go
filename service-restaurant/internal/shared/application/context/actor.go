// Package context contiene el actor validado que transporta identidad y ámbito.
package context

// Actor es un objeto puro con la identidad ya verificada (nunca un header crudo).
type Actor struct {
	UserID      string
	SedeID      string
	Permissions []string
}
