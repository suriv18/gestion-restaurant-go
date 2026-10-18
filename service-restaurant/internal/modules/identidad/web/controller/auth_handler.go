// Package controller expone los casos de uso de identidad por HTTP.
package controller

import (
	"net/http"

	"service-restaurant/internal/modules/identidad/application/dto/command"
	"service-restaurant/internal/modules/identidad/application/dto/query"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	"service-restaurant/internal/modules/identidad/web/dto/request"
	webmapper "service-restaurant/internal/modules/identidad/web/mapper"
	"service-restaurant/internal/platform/web/middleware"
	webrequest "service-restaurant/internal/platform/web/request"
	"service-restaurant/internal/platform/web/response"
)

// AuthHandler: login, logout y consulta de sesión.
type AuthHandler struct {
	iniciar inbound.IniciarSesion
	cerrar  inbound.CerrarSesion
	sesion  inbound.ObtenerSesion
}

func NewAuthHandler(i inbound.IniciarSesion, c inbound.CerrarSesion, s inbound.ObtenerSesion) *AuthHandler {
	return &AuthHandler{iniciar: i, cerrar: c, sesion: s}
}

// Login: POST /auth/login. Nunca cachea la respuesta (lleva el token).
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var body request.Login
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.iniciar.Execute(r.Context(), webmapper.ToIniciarSesion(body))
	if err != nil {
		return err
	}
	if !res.IsValid() {
		return errResultadoInvalido
	}
	if res.Err() != nil {
		return res.Err()
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, http.StatusOK, webmapper.ToTokenResponse(res.Value()))
	return nil
}

// Logout: POST /auth/logout. Revoca la sesión del actor autenticado.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	actor, ok := middleware.ActorFrom(r.Context())
	if !ok {
		return mapper.NoAutenticado()
	}
	res, err := h.cerrar.Execute(r.Context(), command.CerrarSesion{SesionID: actor.SesionID})
	if err != nil {
		return err
	}
	if !res.IsValid() {
		return errResultadoInvalido
	}
	if res.Err() != nil {
		return res.Err()
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Sesion: GET /sesion.
func (h *AuthHandler) Sesion(w http.ResponseWriter, r *http.Request) error {
	actor, ok := middleware.ActorFrom(r.Context())
	if !ok {
		return mapper.NoAutenticado()
	}
	res, err := h.sesion.Execute(r.Context(), query.ObtenerSesion{Actor: actor})
	if err != nil {
		return err
	}
	if !res.IsValid() {
		return errResultadoInvalido
	}
	if res.Err() != nil {
		return res.Err()
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, http.StatusOK, webmapper.ToSesionResponse(res.Value()))
	return nil
}
