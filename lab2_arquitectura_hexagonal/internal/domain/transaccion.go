package domain

import (
	"errors"
	"regexp"
	"strings"
)

// EstadoTransaccion representa los estados válidos en la máquina de estados del dominio.
type EstadoTransaccion string

const (
	EstadoPendiente EstadoTransaccion = "PENDIENTE"
	EstadoAprobada  EstadoTransaccion = "APROBADA"
	EstadoRechazada EstadoTransaccion = "RECHAZADA"
)

var (
	// regexE164 valida el estándar internacional ITU-T E.164.
	regexE164 = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

	ErrIDVacio             = errors.New("el identificador de transacción no puede estar vacío")
	ErrIdempotencyKeyVacia = errors.New("la clave de idempotencia no puede estar vacía")
	ErrMSISDNInvalido      = errors.New("el número telefónico no cumple con el estándar ITU-T E.164")
	ErrMontoInvalido       = errors.New("el monto en centavos debe ser estrictamente positivo")
	ErrTransicionInvalida  = errors.New("transición de estado no permitida: la transacción ya fue finalizada")
)

// TransaccionRecarga es la entidad raíz del dominio puro (Inestabilidad I = 0).
type TransaccionRecarga struct {
	ID             string
	IdempotencyKey string
	MSISDN         string
	MontoCentavos  int64
	Estado         EstadoTransaccion
}

// NuevaTransaccionRecarga construye una entidad válida en estado PENDIENTE garantizando sus invariantes.
func NuevaTransaccionRecarga(id, idempKey, msisdn string, montoCentavos int64) (*TransaccionRecarga, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrIDVacio
	}
	if strings.TrimSpace(idempKey) == "" {
		return nil, ErrIdempotencyKeyVacia
	}
	if !regexE164.MatchString(msisdn) {
		return nil, ErrMSISDNInvalido
	}
	if montoCentavos <= 0 {
		return nil, ErrMontoInvalido
	}

	return &TransaccionRecarga{
		ID:             id,
		IdempotencyKey: idempKey,
		MSISDN:         msisdn,
		MontoCentavos:  montoCentavos,
		Estado:         EstadoPendiente,
	}, nil
}

// Aprobar transiciona la transacción de PENDIENTE a APROBADA.
func (t *TransaccionRecarga) Aprobar() error {
	if t.Estado != EstadoPendiente {
		return ErrTransicionInvalida
	}
	t.Estado = EstadoAprobada
	return nil
}

// Rechazar transiciona la transacción de PENDIENTE a RECHAZADA.
func (t *TransaccionRecarga) Rechazar() error {
	if t.Estado != EstadoPendiente {
		return ErrTransicionInvalida
	}
	t.Estado = EstadoRechazada
	return nil
}