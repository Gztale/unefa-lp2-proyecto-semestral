package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gocql/gocql"
	_ "github.com/lib/pq"
)

type OutboxPayload struct {
	MSISDN          string `json:"msisdn"`
	BucketFecha     string `json:"bucket_fecha"`
	TimestampEvento string `json:"timestamp_evento"`
	IdCDR           string `json:"id_cdr"`
	Celda5G         string `json:"celda_5g"`
	BytesConsumidos int64  `json:"bytes_consumidos"`
	CostoCentavos   int64  `json:"costo_centavos"`
}

func registrarConsumoConOutbox(ctx context.Context, db *sql.DB, msisdn string, costoCentavos, bytesConsumidos int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var abonadoID int64
	err = tx.QueryRowContext(ctx, `
		UPDATE abonados_bcnf
		SET saldo_centavos = saldo_centavos - $1
		WHERE msisdn = $2 AND saldo_centavos >= $1
		RETURNING abonado_id
	`, costoCentavos, msisdn).Scan(&abonadoID)
	if err != nil {
		return fmt.Errorf("no se pudo debitar saldo para %s: %w", msisdn, err)
	}

	payload := OutboxPayload{
		MSISDN:          msisdn,
		BucketFecha:     time.Now().UTC().Format("2006-01-02"),
		TimestampEvento: time.Now().UTC().Format(time.RFC3339Nano),
		IdCDR:           gocql.TimeUUID().String(),
		Celda5G:         "CCS-NORTE-01",
		BytesConsumidos: bytesConsumidos,
		CostoCentavos:   costoCentavos,
	}

	metaJSON, err := json.Marshal(map[string]any{
		"antena_5g": payload.Celda5G,
		"roaming":   true,
	})
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO transacciones_bss (abonado_id, tipo_operacion, monto_centavos, idempotency_key, metadata_jsonb)
		VALUES ($1, 'CONSUMO_5G', $2, $3, $4)
	`, abonadoID, costoCentavos, payload.IdCDR, string(metaJSON))
	if err != nil {
		return err
	}

	eventJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload_jsonb, status)
		VALUES ('ABONADO_5G', $1, 'CDR_CONSUMO_5G', $2, 'PENDING')
	`, msisdn, string(eventJSON))
	if err != nil {
		return err
	}

	return tx.Commit()
}

func procesarLoteOutbox(ctx context.Context, db *sql.DB, scylla *gocql.Session, workerID int) int {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[Worker %d] Error iniciando tx outbox: %v", workerID, err)
		return 0
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT event_id, payload_jsonb
		FROM outbox_events
		WHERE status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT 50
		FOR UPDATE SKIP LOCKED
	`)
	if err != nil {
		log.Printf("[Worker %d] Error consultando outbox: %v", workerID, err)
		return 0
	}
	defer rows.Close()

	type item struct {
		eventID string
		rawJSON []byte
	}
	var lote []item

	for rows.Next() {
		var it item
		if err := rows.Scan(&it.eventID, &it.rawJSON); err != nil {
			log.Printf("[Worker %d] Error escaneando fila: %v", workerID, err)
			continue
		}
		lote = append(lote, it)
	}

	if len(lote) == 0 {
		return 0
	}

	procesados := 0
	for _, it := range lote {
		var p OutboxPayload
		if err := json.Unmarshal(it.rawJSON, &p); err != nil {
			log.Printf("[Worker %d] JSON invalido en evento %s: %v", workerID, it.eventID, err)
			continue
		}

		tsEvento, err := time.Parse(time.RFC3339Nano, p.TimestampEvento)
		if err != nil {
			tsEvento = time.Now().UTC()
		}
		cdrUUID, err := gocql.ParseUUID(p.IdCDR)
		if err != nil {
			cdrUUID = gocql.TimeUUID()
		}

		err = scylla.Query(`
			INSERT INTO cdr_por_abonado_dia
				(msisdn, bucket_fecha, timestamp_evento, id_cdr, celda_5g, bytes_consumidos, costo_centavos)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, p.MSISDN, p.BucketFecha, tsEvento, cdrUUID, p.Celda5G, p.BytesConsumidos, p.CostoCentavos).
			Consistency(gocql.One).
			Exec()

		if err != nil {
			log.Printf("[Worker %d] Fallo proyectando a ScyllaDB evento %s: %v", workerID, it.eventID, err)
			continue
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE outbox_events
			SET status = 'PROCESSED',
			    processed_at = NOW()
			WHERE event_id = $1
		`, it.eventID)
		if err != nil {
			log.Printf("[Worker %d] Error marcando PROCESSED evento %s: %v", workerID, it.eventID, err)
			continue
		}
		procesados++
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[Worker %d] Error en commit outbox: %v", workerID, err)
		return 0
	}
	return procesados
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pgDSN := "host=127.0.0.1 port=5432 user=postgres password=postgres_password dbname=unefa_bss sslmode=disable"
	db, err := sql.Open("postgres", pgDSN)
	if err != nil {
		log.Fatalf("Error conectando a PostgreSQL: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Ping a PostgreSQL fallido: %v", err)
	}

	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "unefa_telco_5g"
	cluster.Consistency = gocql.One
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())

	scylla, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("Error conectando a ScyllaDB: %v", err)
	}
	defer scylla.Close()

	for i := 0; i < 10; i++ {
		if err := registrarConsumoConOutbox(ctx, db, "584120000001", 250, 4096); err != nil {
			log.Printf("Aviso al registrar consumo %d: %v", i, err)
		}
	}

	const numWorkers = 4
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					n := procesarLoteOutbox(ctx, db, scylla, id)
					if n > 0 {
						fmt.Printf("[Worker %d] Proyectados %d eventos a ScyllaDB\n", id, n)
					}
					time.Sleep(200 * time.Millisecond)
				}
			}
		}(w)
	}

	wg.Wait()
	fmt.Println("=== EJERCICIO 4: CICLO OUTBOX COMPLETADO ===")
}
