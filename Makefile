CC = gcc
CFLAGS = -Wall -Wextra -Werror -O0
GO = go

C_SRC = lab1_memoria_y_complejidad/cdr_memory_profiler.c
C_BIN = cdr_profiler.exe
GO_SRC = lab1_memoria_y_complejidad/escape_analyzer.go

.PHONY: all c run-c go-vet clean

all: c

c: $(C_SRC)
	$(CC) $(CFLAGS) $(C_SRC) -o $(C_BIN)

run-c: c
	./$(C_BIN)

go-vet:
	$(GO) vet $(GO_SRC)

clean:
	rm -f $(C_BIN)