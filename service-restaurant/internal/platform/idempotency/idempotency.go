// Package idempotency implementa el registro persistente de idempotencia (ADR-04).
//
// Una clave por intención: repetir la misma solicitud devuelve el efecto ya confirmado;
// reutilizar la clave con otro contenido es un conflicto (409). El registro comparte la
// transacción con el efecto: un rechazo o un fallo hacen rollback y no dejan rastro, de
// modo que el reintento se evalúa de nuevo. Dos solicitudes simultáneas con la misma
// clave se serializan en la base: la segunda espera y repite el resultado de la primera.
package idempotency

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"service-restaurant/internal/platform/transaction"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// DefaultTTL: tiempo durante el cual se recuerda una clave.
const DefaultTTL = 24 * time.Hour

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ParseKey valida y normaliza el valor del encabezado Idempotency-Key (UUID).
func ParseKey(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperror.New(apperror.KindBadRequest, "idempotency_key_requerida", "el encabezado Idempotency-Key es obligatorio")
	}
	if !uuidRe.MatchString(raw) {
		return "", apperror.New(apperror.KindBadRequest, "idempotency_key_invalida", "Idempotency-Key debe ser un UUID")
	}
	return strings.ToLower(raw), nil
}

// Fingerprint es la huella del contenido de la intención: la operación y el cuerpo
// canónico (el que el llamador obtiene al volver a serializar su DTO tipado).
func Fingerprint(operacion string, body []byte) [32]byte {
	h := sha256.New()
	h.Write([]byte(operacion))
	h.Write([]byte{0})
	h.Write(body)
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// Request identifica la intención.
type Request struct {
	Key       string // UUID ya validado con ParseKey
	UsuarioID int64  // la clave está acotada al usuario autenticado
	Operacion string // p. ej. "POST /usuarios"
	Huella    [32]byte
	Status    int           // estado HTTP que se recordará si la operación se confirma
	TTL       time.Duration // 0 = DefaultTTL
	Now       time.Time
}

// Outcome es el resultado de ejecutar (o repetir) la operación.
type Outcome[T any] struct {
	Result   appresult.Result[T]
	Replayed bool
	Status   int
}

var errPrograma = errors.New("idempotency: work devolvió un Result inválido")

// Run ejecuta work una sola vez por (clave, usuario, operación). Solo un resultado
// exitoso se confirma y se recuerda. T debe poder serializarse a JSON.
func Run[T any](
	ctx context.Context,
	db *sql.DB,
	logger *slog.Logger,
	req Request,
	work func(ctx context.Context, tx *sql.Tx) (appresult.Result[T], error),
	options ...transaction.Option,
) (Outcome[T], error) {
	ttl := req.TTL
	if ttl == 0 {
		ttl = DefaultTTL
	}
	out, err := transaction.Run(ctx, db, nil, logger,
		func(ctx context.Context, tx *sql.Tx) (Outcome[T], error) {
			owned, stored, rechazo, err := claim(ctx, tx, req, ttl)
			if err != nil {
				return Outcome[T]{}, err
			}
			if rechazo != nil {
				return Outcome[T]{Result: appresult.Fail[T](rechazo), Status: req.Status}, nil
			}
			if !owned {
				var v T
				if err := json.Unmarshal(stored.body, &v); err != nil {
					return Outcome[T]{}, fmt.Errorf("idempotency: respuesta guardada ilegible: %w", err)
				}
				return Outcome[T]{Result: appresult.Ok(v), Replayed: true, Status: stored.status}, nil
			}
			res, err := work(ctx, tx)
			if err != nil {
				return Outcome[T]{}, err
			}
			if !res.IsValid() {
				return Outcome[T]{}, errPrograma
			}
			if res.IsOk() {
				body, err := json.Marshal(res.Value())
				if err != nil {
					return Outcome[T]{}, fmt.Errorf("idempotency: serializar respuesta: %w", err)
				}
				if err := complete(ctx, tx, req, req.Status, body); err != nil {
					return Outcome[T]{}, err
				}
			}
			return Outcome[T]{Result: res, Status: req.Status}, nil
		},
		func(o Outcome[T]) bool { return o.Result.IsOk() },
		options...,
	)
	return out, err
}

type stored struct {
	status int
	body   []byte
}

// claim reserva la clave dentro de la transacción. owned=false significa que ya existe un
// resultado confirmado para esta misma intención (se repite). Un contenido distinto con la
// misma clave devuelve un conflicto.
func claim(ctx context.Context, tx *sql.Tx, req Request, ttl time.Duration) (owned bool, s stored, rechazo *apperror.AppError, err error) {
	res, err := tx.ExecContext(ctx, `
		INSERT INTO sch_restaurante.idempotencia (clave, usuario_id, operacion, huella, estado, creada_en, expira_en)
		VALUES ($1::uuid, $2, $3, $4, 'EN_CURSO', $5, $6)
		ON CONFLICT (clave, usuario_id, operacion) DO NOTHING`,
		req.Key, req.UsuarioID, req.Operacion, req.Huella[:], req.Now, req.Now.Add(ttl))
	if err != nil {
		return false, stored{}, nil, fmt.Errorf("idempotency: reservar: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return true, stored{}, nil, nil
	}

	// Ya existía (confirmada por otra transacción: ON CONFLICT esperó a que terminara).
	var huella []byte
	var estado string
	var status sql.NullInt32
	var body []byte
	var expira time.Time
	if err := tx.QueryRowContext(ctx, `
		SELECT huella, estado, respuesta_status, respuesta_cuerpo, expira_en
		FROM sch_restaurante.idempotencia
		WHERE clave = $1::uuid AND usuario_id = $2 AND operacion = $3
		FOR UPDATE`, req.Key, req.UsuarioID, req.Operacion).
		Scan(&huella, &estado, &status, &body, &expira); err != nil {
		return false, stored{}, nil, fmt.Errorf("idempotency: leer: %w", err)
	}
	if !req.Now.Before(expira) {
		// Caducada: la clave se reutiliza como una intención nueva.
		if _, err := tx.ExecContext(ctx, `
			UPDATE sch_restaurante.idempotencia
			SET huella = $4, estado = 'EN_CURSO', respuesta_status = NULL, respuesta_cuerpo = NULL,
			    creada_en = $5, expira_en = $6
			WHERE clave = $1::uuid AND usuario_id = $2 AND operacion = $3`,
			req.Key, req.UsuarioID, req.Operacion, req.Huella[:], req.Now, req.Now.Add(ttl)); err != nil {
			return false, stored{}, nil, fmt.Errorf("idempotency: renovar: %w", err)
		}
		return true, stored{}, nil, nil
	}
	if string(huella) != string(req.Huella[:]) {
		return false, stored{}, apperror.New(apperror.KindConflict, "idempotency_key_reutilizada",
			"la clave de idempotencia ya se usó con un contenido distinto"), nil
	}
	if estado != "COMPLETADA" || !status.Valid {
		return false, stored{}, apperror.New(apperror.KindConflict, "idempotency_en_curso",
			"la operación con esta clave sigue en curso"), nil
	}
	return false, stored{status: int(status.Int32), body: body}, nil, nil
}

func complete(ctx context.Context, tx *sql.Tx, req Request, status int, body []byte) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE sch_restaurante.idempotencia
		SET estado = 'COMPLETADA', respuesta_status = $4, respuesta_cuerpo = $5
		WHERE clave = $1::uuid AND usuario_id = $2 AND operacion = $3`,
		req.Key, req.UsuarioID, req.Operacion, status, body)
	if err != nil {
		return fmt.Errorf("idempotency: completar: %w", err)
	}
	return nil
}
