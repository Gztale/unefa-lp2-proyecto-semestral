CC = gcc
CFLAGS = -Wall -Wextra -Werror -O0
GO = go
NODE = node

C_SRC = lab1_memoria_y_complejidad/cdr_memory_profiler.c
C_BIN = cdr_profiler.exe
GO_SRC = lab1_memoria_y_complejidad/escape_analyzer.go

LAB2_DIR = lab2_arquitectura_hexagonal
LAB2_MAIN = $(LAB2_DIR)/cmd/api/main.go
LAB2_BIN = $(LAB2_DIR)/api.exe

LAB3_DIR = lab3_frontend_reactivo
LAB3_DOC = $(LAB3_DIR)/comparativa_web_vitals.md

# lab4_backend_poliglota no realizado por falta de tiempo

LAB5_DIR = lab5_ddd_grpc_y_resiliencia
LAB5_SAGA = $(LAB5_DIR)/cmd/bss_saga_engine.go
LAB5_PROTO = $(LAB5_DIR)/proto/billing_bss.proto

LAB6_DIR = lab6_postgresql_bcnf_mvcc_indices
LAB6_SCHEMA = $(LAB6_DIR)/01_schema_bcnf_y_outbox.sql
LAB6_AUDIT = $(LAB6_DIR)/02_auditoria_mvcc_e_indices.sql
LAB6_REPORT = $(LAB6_DIR)/reporte_explain_buffers.md

LAB7_DIR = lab7_scylladb_cql_benchmark
LAB7_BENCH = $(LAB7_DIR)/cmd/scylla_bench.go
LAB7_CQL = $(LAB7_DIR)/esquema_telco_5g.cql

LAB8_DIR = lab8_transactional_outbox_cqrs
LAB8_WORKER = $(LAB8_DIR)/cmd/polyglot_outbox_worker.go

.PHONY: all lab1 c run-c go-vet lab2 run-lab2 lab3 lab5 vet-lab5 lab6 lab7 vet-lab7 lab8 run-lab8 test-race-lab8 clean

all: lab1 lab2 lab3 lab5 lab6 lab7 lab8

lab1: c go-vet

c: $(C_SRC)
	$(CC) $(CFLAGS) $(C_SRC) -o $(C_BIN)

run-c: c
	./$(C_BIN)

go-vet:
	$(GO) vet $(GO_SRC)

lab2:
	$(GO) vet ./$(LAB2_DIR)/...

run-lab2:
	$(GO) run $(LAB2_MAIN)

lab3:
	@test -f $(LAB3_DOC) && echo "Lab 3 OK"

lab5: vet-lab5

vet-lab5:
	$(GO) vet $(LAB5_SAGA)

lab6:
	@test -f $(LAB6_SCHEMA) && test -f $(LAB6_AUDIT) && test -f $(LAB6_REPORT) && echo "Lab 6 OK"

lab7: vet-lab7

vet-lab7:
	$(GO) vet $(LAB7_BENCH)

lab8: run-lab8

run-lab8:
	$(GO) run -race $(LAB8_WORKER)

test-race-lab8:
	$(GO) test -race ./...

clean:
	if [ -f $(LAB2_BIN) ]; then rm -f $(LAB2_BIN); fi
	if [ -f $(C_BIN) ]; then rm -f $(C_BIN); fi