package adapters

import (
	"context"
	"sync"

	"lab2_arquitectura_hexagonal/internal/application"
	"lab2_arquitectura_hexagonal/internal/domain"
	"lab2_arquitectura_hexagonal/internal/ports"
)

type RepositorioMemoria struct {
	mu          sync.RWMutex
	porID       map[string]*domain.TransaccionRecarga
	porIdempKey map[string]*domain.TransaccionRecarga
}

func NuevoRepositorioMemoria() *RepositorioMemoria {
	return &RepositorioMemoria{
		porID:       make(map[string]*domain.TransaccionRecarga),
		porIdempKey: make(map[string]*domain.TransaccionRecarga),
	}
}

func (r *RepositorioMemoria) BuscarPorIdempotencyKey(_ context.Context, key string) (*domain.TransaccionRecarga, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tx, ok := r.porIdempKey[key]
	if !ok {
		return nil, nil
	}
	return tx, nil
}

func (r *RepositorioMemoria) Guardar(_ context.Context, tx *domain.TransaccionRecarga) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.porID[tx.ID] = tx
	if tx.IdempotencyKey != "" {
		r.porIdempKey[tx.IdempotencyKey] = tx
	}
	return nil
}

type PasarelaSimulada struct {
	mu          sync.Mutex
	CobrosCount int
}

func NuevaPasarelaSimulada() *PasarelaSimulada {
	return &PasarelaSimulada{}
}

func (p *PasarelaSimulada) Debitar(_ context.Context, _ *domain.TransaccionRecarga) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.CobrosCount++
	return true, nil
}

type AuditoriaConsola struct {
	mu      sync.Mutex
	Eventos []string
}

func NuevaAuditoriaConsola() *AuditoriaConsola {
	return &AuditoriaConsola{
		Eventos: make([]string, 0),
	}
}

func (a *AuditoriaConsola) RegistrarEvento(_ context.Context, evento string, tx *domain.TransaccionRecarga) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.Eventos = append(a.Eventos, evento+":"+tx.ID)
	return nil
}

type ControladorRecarga struct {
	servicio *application.ServicioRecarga
}

func NuevoControladorRecarga(servicio *application.ServicioRecarga) *ControladorRecarga {
	return &ControladorRecarga{servicio: servicio}
}

func (c *ControladorRecarga) Procesar(ctx context.Context, req application.SolicitudRecarga) (*domain.TransaccionRecarga, error) {
	return c.servicio.EjecutarRecarga(ctx, req)
}

var (
	_ ports.RepositorioTransacciones = (*RepositorioMemoria)(nil)
	_ ports.PasarelaPagoBancaria     = (*PasarelaSimulada)(nil)
	_ ports.NotificadorAuditoria     = (*AuditoriaConsola)(nil)
)