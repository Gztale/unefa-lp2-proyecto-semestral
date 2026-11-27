CC = gcc
CFLAGS = -Wall -Wextra -Werror -O0
GO = go
NODE = node

C_SRC = lab1_memoria_y_complejidad/cdr_memory_profiler.c
C_BIN = cdr_profiler.exe
GO_SRC = lab1_memoria_y_complejidad/escape_analyzer.go
LAB2_DIR = lab2_arquitectura_hexagonal
LAB3_DOC = lab3_frontend_reactivo/comparativa_web_vitals.md

.PHONY: all lab1 c run-c go-vet lab2 lab3 clean

all: lab1 lab2 lab3

lab1: c go-vet

c: $(C_SRC)
	$(CC) $(CFLAGS) $(C_SRC) -o $(C_BIN)

run-c: c
	./$(C_BIN)

go-vet:
	$(GO) vet $(GO_SRC)

lab2:
	cd $(LAB2_DIR) && $(GO) vet ./... && $(GO) build ./cmd/api

lab3:
	$(NODE) -e "const fs=require('fs'); if(!fs.existsSync('$(LAB3_DOC)')) process.exit(1); console.log('Lab 3 validado: $(LAB3_DOC) presente');"

clean:
	if exist "$(C_BIN)" del /Q "$(C_BIN)"
	if exist "$(LAB2_DIR)\api.exe" del /Q "$(LAB2_DIR)\api.exe"