# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# Imagen minima sin shell, usuario no root. Configuracion solo por variables de entorno.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /app/api
# Las migraciones van embebidas; se ejecutan como paso controlado, no al arrancar cada replica:
#   docker run --rm --entrypoint /app/migrate <imagen> up
COPY --from=build /out/migrate /app/migrate
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
