package main

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/gocql/gocql"
)

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func main() {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "unefa_telco_5g"
	cluster.Consistency = gocql.One
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	cluster.NumConns = 4
	cluster.Timeout = 10 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("Error conectando a ScyllaDB: %v", err)
	}
	defer session.Close()

	const workers = 16
	const opsPorWorker = 500
	totalOps := workers * opsPorWorker

	query := `INSERT INTO cdr_por_abonado_dia
		(msisdn, bucket_fecha, timestamp_evento, id_cdr, celda_5g, bytes_consumidos, costo_centavos)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	var wg sync.WaitGroup
	latenciasCh := make(chan time.Duration, totalOps)

	inicioGlobal := time.Now()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			msisdn := fmt.Sprintf("58414555%04d", workerID)
			bucket := time.Now().UTC().Format("2006-01-02")

			for i := 0; i < opsPorWorker; i++ {
				t0 := time.Now()
				err := session.Query(
					query,
					msisdn,
					bucket,
					time.Now().UTC(),
					gocql.TimeUUID(),
					fmt.Sprintf("NR-CELL-%d", i%16),
					int64((i+1)*1024),
					int64(50),
				).Exec()
				lat := time.Since(t0)

				if err != nil {
					log.Printf("Worker %d error insertando: %v", workerID, err)
					continue
				}
				latenciasCh <- lat
			}
		}(w)
	}

	wg.Wait()
	close(latenciasCh)

	duracionTotal := time.Since(inicioGlobal)

	var latencias []time.Duration
	for l := range latenciasCh {
		latencias = append(latencias, l)
	}
	sort.Slice(latencias, func(i, j int) bool { return latencias[i] < latencias[j] })

	completadas := len(latencias)
	throughput := float64(completadas) / duracionTotal.Seconds()

	p50 := percentile(latencias, 0.50)
	p95 := percentile(latencias, 0.95)
	p99 := percentile(latencias, 0.99)

	fmt.Println("=== EJERCICIO 3: BENCHMARK SCYLLADB (TOKEN-AWARE, W=16) ===")
	fmt.Printf("Operaciones completadas : %d / %d\n", completadas, totalOps)
	fmt.Printf("Tiempo total            : %v\n", duracionTotal)
	fmt.Printf("Throughput              : %.2f ops/s\n", throughput)
	fmt.Printf("Latencia P50            : %d us (%.3f ms)\n", p50.Microseconds(), float64(p50.Microseconds())/1000.0)
	fmt.Printf("Latencia P95            : %d us (%.3f ms)\n", p95.Microseconds(), float64(p95.Microseconds())/1000.0)
	fmt.Printf("Latencia P99            : %d us (%.3f ms)\n", p99.Microseconds(), float64(p99.Microseconds())/1000.0)
}
