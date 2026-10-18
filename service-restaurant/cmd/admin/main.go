// Comando admin: instalación inicial. Crea el primer administrador (rol ADMIN) y, si aún no
// existen, el negocio y la sucursal. Se ejecuta una vez por el instalador; se niega a correr
// si ya existe un usuario humano.
//
// La contraseña se lee de ADMIN_PASSWORD o, si no está, de la primera línea de stdin
// (nunca por argumento, para no dejarla en el historial ni en la lista de procesos).
//
//	echo 'MiClaveSegura2026' | admin -username admin -ruc 20123456789 \
//	  -razon-social "Mi Restaurante SAC" -nombre-comercial "Mi Restaurante" \
//	  -direccion-fiscal "Av. Principal 123" -sucursal-codigo PRINCIPAL \
//	  -sucursal-nombre "Local Principal" -sucursal-direccion "Av. Principal 123"
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"service-restaurant/internal/bootstrap"
	"service-restaurant/internal/platform/config"
	"service-restaurant/internal/platform/observability"
	"service-restaurant/internal/shared/kernel/apperror"
)

func main() {
	if err := run(); err != nil {
		var ae *apperror.AppError
		if errors.As(err, &ae) {
			fmt.Fprintf(os.Stderr, "error [%s]: %s\n", ae.Code, ae.Message)
		} else {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(1)
	}
}

func run() error {
	var p bootstrap.AdminParams
	fs := flag.NewFlagSet("admin", flag.ContinueOnError)
	fs.StringVar(&p.Username, "username", "", "nombre de acceso del administrador")
	fs.StringVar(&p.RUC, "ruc", "", "RUC del negocio (11 dígitos)")
	fs.StringVar(&p.RazonSocial, "razon-social", "", "razón social")
	fs.StringVar(&p.NombreComercial, "nombre-comercial", "", "nombre comercial")
	fs.StringVar(&p.DireccionFiscal, "direccion-fiscal", "", "dirección fiscal")
	fs.StringVar(&p.SucursalCodigo, "sucursal-codigo", "", "código de la sucursal")
	fs.StringVar(&p.SucursalNombre, "sucursal-nombre", "", "nombre de la sucursal")
	fs.StringVar(&p.SucursalDireccion, "sucursal-direccion", "", "dirección de la sucursal")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	p.Password = os.Getenv("ADMIN_PASSWORD")
	if p.Password == "" {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return errors.New("falta la contraseña: defina ADMIN_PASSWORD o envíela por stdin")
		}
		p.Password = strings.TrimRight(line, "\r\n")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := observability.NewLogger(cfg.LogLevel)
	r, err := bootstrap.InstalarAdministrador(context.Background(), cfg, logger, p)
	if err != nil {
		return err
	}
	fmt.Printf("administrador creado: usuario_id=%d sucursal_id=%d\n", r.UsuarioID, r.SucursalID)
	return nil
}
