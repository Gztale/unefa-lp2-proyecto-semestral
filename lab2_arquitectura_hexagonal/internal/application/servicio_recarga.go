package application

import (
	"context"
	"errors"

	"lab2_arquitectura_hexagonal/internal/domain"
	"lab2_arquitectura_hexagonal/internal/ports"
)

var (
	ErrDebitoRechazado = errors.New("la pasarela bancaria rechazó el débito de la recarga")
)

type SolicitudRecarga struct {
	ID             string
	IdempotencyKey string
	MSISDN         string
	MontoCentavos  int64
}

type ServicioRecarga struct {
	repo      ports.RepositorioTransacciones
	pasarela  ports.PasarelaPagoBancaria
	auditoria ports.NotificadorAuditoria
}

func NuevoServicioRecarga(
	repo ports.RepositorioTransacciones,
	pasarela ports.PasarelaPagoBancaria,
	auditoria ports.NotificadorAuditoria,
) *ServicioRecarga {
	return &ServicioRecarga{
		repo:      repo,
		pasarela:  pasarela,
		auditoria: auditoria,
	}
}

func (s *ServicioRecarga) EjecutarRecarga(ctx context.Context, req SolicitudRecarga) (*domain.TransaccionRecarga, error) {
	existente, err := s.repo.BuscarPorIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existente != nil {
		_ = s.auditoria.RegistrarEvento(ctx, "RECARGA_IDEMPOTENTE_RETORNADA", existente)
		return existente, nil
	}

	tx, err := domain.NuevaTransaccionRecarga(req.ID, req.IdempotencyKey, req.MSISDN, req.MontoCentavos)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Guardar(ctx, tx); err != nil {
		return nil, err
	}
	_ = s.auditoria.RegistrarEvento(ctx, "RECARGA_PENDIENTE", tx)

	aprobado, err := s.pasarela.Debitar(ctx, tx)
	if err != nil {
		_ = tx.Rechazar()
		_ = s.repo.Guardar(ctx, tx)
		_ = s.auditoria.RegistrarEvento(ctx, "RECARGA_ERROR_PASARELA", tx)
		return nil, err
	}

	if !aprobado {
		_ = tx.Rechazar()
		_ = s.repo.Guardar(ctx, tx)
		_ = s.auditoria.RegistrarEvento(ctx, "RECARGA_RECHAZADA", tx)
		return tx, ErrDebitoRechazado
	}

	if err := tx.Aprobar(); err != nil {
		return nil, err
	}

	if err := s.repo.Guardar(ctx, tx); err != nil {
		return nil, err
	}

	_ = s.auditoria.RegistrarEvento(ctx, "RECARGA_APROBADA", tx)
	return tx, nil
}