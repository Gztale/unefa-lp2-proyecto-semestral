package main

import "fmt"

type Record struct {
	ID    uint64
	Score int
}

// stayOnStack: el valor se copia por retorno; no se devuelve su dirección.
func stayOnStack() int {
	localVal := 42
	return localVal
}

// escapeByPointer: devolver el puntero a una variable local fuerza su escape al heap.
func escapeByPointer() *Record {
	rec := Record{ID: 1001, Score: 95}
	return &rec
}

// escapeByInterface: pasar un valor a una interfaz (como fmt.Println) suele hacerlo escapar al heap.
func escapeByInterface() {
	msg := "escape check"
	fmt.Println(msg)
}

func main() {
	a := stayOnStack()
	b := escapeByPointer()
	escapeByInterface()
	fmt.Println(a, b.ID, b.Score)
}