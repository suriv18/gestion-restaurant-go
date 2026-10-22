# service-restaurant

Backend Go (monolito modular) del sistema de restaurante. Estructura y arquitectura según `../docs/estructura/PLANTILLA_BACKEND_GO.md`; diseño en `docs/ADD.md`, `docs/module-map.md` y `docs/adr/`.

Estado: núcleo + migraciones por dominio + módulos `identidad` (I1: acceso, autorización y administración de accesos; I2: negocio y sucursales) y `atencion` (I2: zonas y mesas). El módulo `catalogo` es el esqueleto de ejemplo de la plantilla.

## Levantar con Docker

Requisitos: Docker. Desde esta carpeta:

```bash
# 1. Secreto de los JWT (>= 32 bytes). Se guarda en .env, que git ignora.
echo "JWT_SECRET=$(openssl rand -base64 48)" > .env
echo "API_PORT=8080" >> .env        # puerto del host; cámbielo si 8080 está ocupado

# 2. PostgreSQL 18 + migraciones + API
docker compose up -d --build

# 3. Instalación inicial (una sola vez): primer administrador, negocio y sucursal
echo 'UnaClaveSegura2026' | docker compose run --rm -T admin \
  -username admin -ruc 20123456789 -razon-social "Mi Restaurante SAC" \
  -nombre-comercial "Mi Restaurante" -direccion-fiscal "Av. Principal 123" \
  -sucursal-codigo PRINCIPAL -sucursal-nombre "Local Principal" -sucursal-direccion "Av. Principal 123"
```

La contraseña se lee de `ADMIN_PASSWORD` o de stdin (nunca por argumento). `admin` se niega a correr si ya existe un usuario humano.

Comprobación:

```bash
curl localhost:8080/readyz
curl -X POST localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"UnaClaveSegura2026"}'
```

`docker compose down` detiene el stack y conserva los datos; `docker compose down -v` también borra la base.

## Desarrollo

```bash
docker compose up -d db                 # solo PostgreSQL 18
export DATABASE_URL='postgres://restaurant:change-me@localhost:5432/restaurant?sslmode=disable'
go run ./cmd/migrate up                 # migraciones (también: down, status)
JWT_SECRET=... go run ./cmd/api         # API

go vet ./... && go test ./...           # unitarias y arquitectura
go test ./tests/integration/...         # integración: PostgreSQL 18 con testcontainers (requiere Docker)
```

## API

Base `/api/v1`: `POST /auth/login`, `POST /auth/logout`, `GET /sesion`, y la administración de `roles`, `permisos` y `usuarios` — incluidos `GET /usuarios` (el equipo con sus roles y sedes) y `GET /sucursales-asignables` — (requiere `USUARIO_ADMINISTRAR`; las altas, bajas y restauraciones exigen `Idempotency-Key` UUID). Configuración y salón (I2): `GET/PATCH /negocio`, `GET/POST /sucursales`, `GET/PATCH /sucursales/{id}`, `/zonas…` y `/mesas…` (con sus `…/baja` y `…/restauracion`). Permisos nuevos: `NEGOCIO_VER`, `NEGOCIO_ADMINISTRAR`, `SUCURSAL_VER`, `SUCURSAL_ADMINISTRAR`, `MESA_VER`, `MESA_ADMINISTRAR`. La baja de mesa exige motivo: `docs/adr/005-baja-de-mesa-con-motivo.md`. Contrato propuesto en `../docs/planificacion/03_api/openapi.yaml`; desviaciones en `docs/ADD.md` §8. Sondas: `GET /healthz`, `GET /readyz`.
