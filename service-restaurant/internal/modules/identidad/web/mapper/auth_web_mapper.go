// Package mapper convierte entre DTO web y DTO de aplicación.
package mapper

import (
	"strconv"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/web/dto/request"
	"service-restaurant/internal/modules/identidad/web/dto/response"
)

func ToIniciarSesion(r request.Login) cmd.IniciarSesion {
	return cmd.IniciarSesion{Username: r.Username, Password: r.Password}
}

func ToTokenResponse(r dtoresult.SesionIniciada) response.Token {
	return response.Token{AccessToken: r.AccessToken, TokenType: "Bearer", ExpiresIn: r.ExpiresIn}
}

func ToSesionResponse(r dtoresult.SesionActual) response.SesionActual {
	sedes := make([]string, len(r.Sucursales))
	for i, s := range r.Sucursales {
		sedes[i] = strconv.FormatInt(s, 10)
	}
	permisos := r.Permisos
	if permisos == nil {
		permisos = []string{}
	}
	return response.SesionActual{
		UsuarioID: strconv.FormatInt(r.UsuarioID, 10), Username: r.Username,
		Sucursales: sedes, Permisos: permisos,
	}
}
