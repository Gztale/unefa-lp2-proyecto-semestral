package main

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

var (
	ErrSaldoInsuficiente   = errors.New("saldo insuficiente para completar la operacion")
	ErrMonedaIncompatible  = errors.New("las monedas no coinciden")
	ErrSuscripcionInactiva = errors.New("la suscripcion 5G no esta activa")
	ErrCircuitoAbierto     = errors.New("circuit breaker en estado OPEN")
)

// Dinero es un Value Object inmutable expresado en centavos (int64).
type Dinero struct {
	MontoCentavos int64
	Moneda        string
}

func NuevoDinero(montoCentavos int64, moneda string) (Dinero, error) {
	if montoCentavos < 0 {
		return Dinero{}, ErrSaldoInsuficiente
	}
	return Dinero{MontoCentavos: montoCentavos, Moneda: moneda}, nil
}

func (d Dinero) Sumar(otro Dinero) (Dinero, error) {
	if d.Moneda != otro.Moneda {
		return Dinero{}, ErrMonedaIncompatible
	}
	return Dinero{
		MontoCentavos: d.MontoCentavos + otro.MontoCentavos,
		Moneda:        d.Moneda,
	}, nil
}

func (d Dinero) Restar(otro Dinero) (Dinero, error) {
	if d.Moneda != otro.Moneda {
		return Dinero{}, ErrMonedaIncompatible
	}
	nuevo := d.MontoCentavos - otro.MontoCentavos
	if nuevo < 0 {
		return Dinero{}, ErrSaldoInsuficiente
	}
	return Dinero{
		MontoCentavos: nuevo,
		Moneda:        d.Moneda,
	}, nil
}

// CargoAplicadoEvent es un evento de dominio inmutable.
type CargoAplicadoEvent struct {
	EventID        string
	IdempotencyKey string
	SuscripcionID  string
	MSISDN         string
	CargoCentavos  int64
	SaldoRestante  int64
	Moneda         string
	OcurridoEn     time.Time
}

// Suscripcion5G es la Raíz de Agregado que protege las invariantes de facturación.
type Suscripcion5G struct {
	ID                         string
	MSISDN                     string
	Activa                     bool
	Saldo                      Dinero
	EventosNoConf              []CargoAplicadoEvent
	clavesProcesadas           map[string]CargoAplicadoEvent
	compensacionesProcesadas map[string]bool
}

func NuevaSuscripcion5G(id, msisdn string, saldoInicial Dinero) *Suscripcion5G {
	return &Suscripcion5G{
		ID:                         id,
		MSISDN:                     msisdn,
		Activa:                     true,
		Saldo:                      saldoInicial,
		clavesProcesadas:           make(map[string]CargoAplicadoEvent),
		compensacionesProcesadas: make(map[string]bool),
	}
}

func (s *Suscripcion5G) AplicarCargo(idempotencyKey string, cargo Dinero) (CargoAplicadoEvent, error) {
	if ev, existe := s.clavesProcesadas[idempotencyKey]; existe {
		return ev, nil
	}
	if !s.Activa {
		return CargoAplicadoEvent{}, ErrSuscripcionInactiva
	}
	nuevoSaldo, err := s.Saldo.Restar(cargo)
	if err != nil {
		return CargoAplicadoEvent{}, err
	}
	s.Saldo = nuevoSaldo

	ev := CargoAplicadoEvent{
		EventID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		IdempotencyKey: idempotencyKey,
		SuscripcionID:  s.ID,
		MSISDN:         s.MSISDN,
		CargoCentavos:  cargo.MontoCentavos,
		SaldoRestante:  s.Saldo.MontoCentavos,
		Moneda:         s.Saldo.Moneda,
		OcurridoEn:     time.Now().UTC(),
	}
	s.clavesProcesadas[idempotencyKey] = ev
	s.EventosNoConf = append(s.EventosNoConf, ev)
	return ev, nil
}

func (s *Suscripcion5G) Acreditar(idempotencyKey string, monto Dinero) error {
	if idempotencyKey != "" && s.compensacionesProcesadas[idempotencyKey] {
		return nil
	}
	if !s.Activa {
		return ErrSuscripcionInactiva
	}
	nuevoSaldo, err := s.Saldo.Sumar(monto)
	if err != nil {
		return err
	}
	s.Saldo = nuevoSaldo
	if idempotencyKey != "" {
		s.compensacionesProcesadas[idempotencyKey] = true
	}
	return nil
}

// EstadoCB representa los estados del autómata Circuit Breaker.
type EstadoCB string

const (
	EstadoClosed   EstadoCB = "CLOSED"
	EstadoOpen     EstadoCB = "OPEN"
	EstadoHalfOpen EstadoCB = "HALF-OPEN"
)

// CircuitBreaker protege llamadas externas de forma thread-safe.
type CircuitBreaker struct {
	mu               sync.Mutex
	estado           EstadoCB
	fallosConsecutivos int
	umbralFallos     int
	tiempoEnAbierto  time.Duration
	abiertoDesde     time.Time
	rng              *rand.Rand
}

func NuevoCircuitBreaker(umbralFallos int, tiempoEnAbierto time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		estado:          EstadoClosed,
		umbralFallos:    umbralFallos,
		tiempoEnAbierto: tiempoEnAbierto,
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (cb *CircuitBreaker) permitir() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.estado {
	case EstadoClosed:
		return true
	case EstadoOpen:
		if time.Since(cb.abiertoDesde) >= cb.tiempoEnAbierto {
			cb.estado = EstadoHalfOpen
			return true
		}
		return false
	case EstadoHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) registrarResultado(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err == nil {
		cb.fallosConsecutivos = 0
		cb.estado = EstadoClosed
		return
	}

	cb.fallosConsecutivos++
	if cb.estado == EstadoHalfOpen || cb.fallosConsecutivos >= cb.umbralFallos {
		cb.estado = EstadoOpen
		cb.abiertoDesde = time.Now()
	}
}

func (cb *CircuitBreaker) EstadoActual() EstadoCB {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.estado
}

// BackoffFullJitter calcula T = Uniform(0, min(T_max, T_base * 2^attempt)).
func BackoffFullJitter(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	tBase := 25 * time.Millisecond
	tMax := 400 * time.Millisecond

	exp := math.Pow(2, float64(attempt))
	candidato := time.Duration(float64(tBase) * exp)
	if candidato > tMax || candidato <= 0 {
		candidato = tMax
	}

	cbMu := rand.Int63n(int64(candidato) + 1)
	return time.Duration(cbMu)
}

func (cb *CircuitBreaker) EjecutarConReintento(maxIntentos int, fn func() error) error {
	for intento := 0; intento < maxIntentos; intento++ {
		if !cb.permitir() {
			return ErrCircuitoAbierto
		}
		err := fn()
		cb.registrarResultado(err)
		if err == nil {
			return nil
		}
		if intento < maxIntentos-1 {
			espera := BackoffFullJitter(intento)
			time.Sleep(espera)
		}
	}
	return errors.New("agotados los reintentos con full jitter")
}

// PasoSaga representa una acción y su compensación idempotente.
type PasoSaga struct {
	Nombre      string
	Ejecutar    func(idempotencyKey string) error
	Compensar   func(idempotencyKey string) error
	ejecutadoOK bool
}

// SagaOrchestrator coordina T1 -> T2 -> T3 y compensa en orden inverso.
type SagaOrchestrator struct {
	pasos []*PasoSaga
}

func NuevoSagaOrchestrator(pasos ...*PasoSaga) *SagaOrchestrator {
	return &SagaOrchestrator{pasos: pasos}
}

func (so *SagaOrchestrator) Ejecutar(sagaID string) error {
	for idx, paso := range so.pasos {
		key := fmt.Sprintf("%s-paso-%d", sagaID, idx+1)
		if err := paso.Ejecutar(key); err != nil {
			fmt.Printf("  [SAGA] Fallo en %s (%v). Iniciando compensaciones...\n", paso.Nombre, err)
			so.compensarDesde(idx-1, sagaID)
			return err
		}
		paso.ejecutadoOK = true
		fmt.Printf("  [SAGA] %s completado (key=%s)\n", paso.Nombre, key)
	}
	return nil
}

func (so *SagaOrchestrator) compensarDesde(ultimoIdx int, sagaID string) {
	for i := ultimoIdx; i >= 0; i-- {
		paso := so.pasos[i]
		if paso.ejecutadoOK && paso.Compensar != nil {
			compKey := fmt.Sprintf("%s-comp-%d", sagaID, i+1)
			if err := paso.Compensar(compKey); err != nil {
				fmt.Printf("  [SAGA-COMP] Error compensando %s: %v\n", paso.Nombre, err)
			} else {
				fmt.Printf("  [SAGA-COMP] Compensado %s (key=%s)\n", paso.Nombre, compKey)
			}
		}
	}
}

func main() {
	fmt.Println("=== EJERCICIO 1: DDD + CIRCUIT BREAKER + FULL JITTER + SAGA ===")

	saldoInicial, _ := NuevoDinero(10000, "USD") // $100.00 en centavos
	sub := NuevaSuscripcion5G("SUB-5G-001", "584145550101", saldoInicial)
	cargo, _ := NuevoDinero(2500, "USD") // $25.00

	cb := NuevoCircuitBreaker(2, 150*time.Millisecond)
	fmt.Printf("Estado inicial CB: %s | Jitter muestra: %v\n", cb.EstadoActual(), BackoffFullJitter(2))

	// Escenario con fallo simulado en T2 (Aprovisionar QoS 5G) -> dispara C1
	fallosQoS := 0
	saga := NuevoSagaOrchestrator(
		&PasoSaga{
			Nombre: "T1: Debitar Billetera",
			Ejecutar: func(key string) error {
				ev, err := sub.AplicarCargo(key, cargo)
				if err != nil {
					return err
				}
				fmt.Printf("    -> Debitado %d centavos. Saldo actual: %d\n", ev.CargoCentavos, ev.SaldoRestante)
				return nil
			},
			Compensar: func(key string) error {
				err := sub.Acreditar(key, cargo)
				// Prueba de idempotencia ante reintento de compensación con la misma clave
				_ = sub.Acreditar(key, cargo)
				fmt.Printf("    -> Reembolsado %d centavos. Saldo restaurado: %d\n", cargo.MontoCentavos, sub.Saldo.MontoCentavos)
				return err
			},
		},
		&PasoSaga{
			Nombre: "T2: Aprovisionar QoS 5G",
			Ejecutar: func(key string) error {
				return cb.EjecutarConReintento(2, func() error {
					fallosQoS++
					return errors.New("timeout en controlador PCF 5G")
				})
			},
			Compensar: func(key string) error {
				fmt.Printf("    -> [C2] Liberando reserva QoS 5G (key=%s)\n", key)
				return nil
			},
		},
		&PasoSaga{
			Nombre: "T3: Emitir Factura",
			Ejecutar: func(key string) error {
				fmt.Println("    -> Factura emitida")
				return nil
			},
			Compensar: func(key string) error {
				fmt.Printf("    -> [C3] Anulando factura emitida (key=%s)\n", key)
				return nil
			},
		},
	)

	err := saga.Ejecutar("SAGA-1001")
	fmt.Printf("Resultado Saga: err=%v | Estado CB tras fallos=%s | Saldo final=%d centavos\n",
		err, cb.EstadoActual(), sub.Saldo.MontoCentavos)
}
