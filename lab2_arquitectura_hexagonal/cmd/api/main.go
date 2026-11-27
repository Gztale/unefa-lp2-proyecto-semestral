package main

import (
	"context"
	"fmt"
	"log"

	"lab2_arquitectura_hexagonal/internal/adapters"
	"lab2_arquitectura_hexagonal/internal/application"
)

func main() {
	ctx := context.Background()

	repo := adapters.NuevoRepositorioMemoria()
	pasarela := adapters.NuevaPasarelaSimulada()
	auditoria := adapters.NuevaAuditoriaConsola()
	servicio := application.NuevoServicioRecarga(repo, pasarela, auditoria)
	controlador := adapters.NuevoControladorRecarga(servicio)

	// 1. Recarga exitosa con número E.164 válido y monto en centavos
	tx1, err := controlador.Procesar(ctx, application.SolicitudRecarga{
		ID:             "tx_001",
		IdempotencyKey: "550e8400-e29b-41d4-a716-446655440000",
		MSISDN:         "+584121234567",
		MontoCentavos:  500,
	})
	if err != nil {
		log.Fatalf("caso 1 inesperado: %v", err)
	}
	fmt.Printf("Caso 1 (Exitosa): ID=%s Estado=%s MontoCentavos=%d\n", tx1.ID, tx1.Estado, tx1.MontoCentavos)

	// 2. Rechazo por número E.164 inválido
	_, err = controlador.Procesar(ctx, application.SolicitudRecarga{
		ID:             "tx_002",
		IdempotencyKey: "550e8400-e29b-41d4-a716-446655440001",
		MSISDN:         "04121234567",
		MontoCentavos:  500,
	})
	fmt.Printf("Caso 2 (E.164 inválido): error=%v\n", err)

	// 3. Tres reintentos con la misma Idempotency-Key -> cobra solo una vez en este caso
	cobrosAntesCaso3 := pasarela.CobrosCount
	for i := 1; i <= 3; i++ {
		txRetry, retryErr := controlador.Procesar(ctx, application.SolicitudRecarga{
			ID:             fmt.Sprintf("tx_retry_%d", i),
			IdempotencyKey: "550e8400-e29b-41d4-a716-446655440099",
			MSISDN:         "+584149876543",
			MontoCentavos:  1000,
		})
		cobrosCaso3 := pasarela.CobrosCount - cobrosAntesCaso3
		fmt.Printf("Caso 3 (Reintento %d): txID=%s estado=%s cobrosCaso3=%d cobrosTotales=%d err=%v\n",
			i, txRetry.ID, txRetry.Estado, cobrosCaso3, pasarela.CobrosCount, retryErr)
	}
}