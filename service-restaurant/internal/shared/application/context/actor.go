// Package context contiene el actor validado que transporta identidad y ámbito.
package context

import "sort"

// Grant es un permiso efectivo en una sucursal.
type Grant struct {
	SucursalID int64
	Permiso    string
}

// Actor es un objeto puro con la identidad ya verificada (nunca un header crudo).
// Es inmutable: los permisos se fijan al construirlo.
type Actor struct {
	UserID   int64
	Username string
	SesionID int64
	grants   map[int64]map[string]struct{}
}

// NewActor construye un actor con sus permisos efectivos por sucursal.
func NewActor(userID int64, username string, sesionID int64, grants []Grant) Actor {
	m := make(map[int64]map[string]struct{})
	for _, g := range grants {
		if m[g.SucursalID] == nil {
			m[g.SucursalID] = make(map[string]struct{})
		}
		m[g.SucursalID][g.Permiso] = struct{}{}
	}
	return Actor{UserID: userID, Username: username, SesionID: sesionID, grants: m}
}

// IsZero indica que no hay un usuario autenticado.
func (a Actor) IsZero() bool { return a.UserID == 0 }

// Can indica si el actor tiene el permiso en esa sucursal.
func (a Actor) Can(permiso string, sucursalID int64) bool {
	_, ok := a.grants[sucursalID][permiso]
	return ok
}

// CanAnywhere indica si el actor tiene el permiso en alguna sucursal.
func (a Actor) CanAnywhere(permiso string) bool {
	for _, ps := range a.grants {
		if _, ok := ps[permiso]; ok {
			return true
		}
	}
	return false
}

// SucursalesCon devuelve, ordenadas, las sucursales donde el actor tiene el permiso.
func (a Actor) SucursalesCon(permiso string) []int64 {
	var out []int64
	for s, ps := range a.grants {
		if _, ok := ps[permiso]; ok {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Sucursales devuelve, ordenadas, todas las sucursales del actor.
func (a Actor) Sucursales() []int64 {
	out := make([]int64, 0, len(a.grants))
	for s := range a.grants {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Permisos devuelve, ordenados y sin repetir, los permisos del actor en cualquier sucursal.
func (a Actor) Permisos() []string {
	set := make(map[string]struct{})
	for _, ps := range a.grants {
		for p := range ps {
			set[p] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
