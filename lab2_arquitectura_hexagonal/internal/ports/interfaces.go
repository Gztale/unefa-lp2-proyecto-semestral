package ports

import (
	"context"

	"lab2_arquitectura_hexagonal/internal/domain"
)

// RepositorioTransacciones define el puerto secundario de persistencia e idempotencia.
type RepositorioTransacciones interface {
	BuscarPorIdempotencyKey(ctx context.Context, key string) (*domain.TransaccionRecarga, error)
	Guardar(ctx context.Context, tx *domain.TransaccionRecarga) error
}

// PasarelaPagoBancaria define el puerto secundario hacia el proveedor de cobro/recarga.
type PasarelaPagoBancaria interface {
	Debitar(ctx context.Context, tx *domain.TransaccionRecarga) (bool, error)
}

// NotificadorAuditoria define el puerto secundario para registrar eventos de auditoría.
type NotificadorAuditoria interface {
	RegistrarEvento(ctx context.Context, evento string, tx *domain.TransaccionRecarga) error
}