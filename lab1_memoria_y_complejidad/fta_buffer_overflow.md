# Fault Tree Analysis (FTA): Buffer Overflow

## 1. Evento Tope (Top Event)
- **TE:** Ocurrencia de un *Buffer Overflow* (escritura más allá de los límites asignados a un búfer en memoria).

## 2. Estructura del Árbol de Fallos
```text
[ EVENTO TOPE: Desbordamiento de Búfer (Buffer Overflow) ]
                                            |
                                    | COMPUERTA OR |
                                    +--------------+
                                    /              \
                                   /                \
        [ G1: Stack Buffer Overflow ]              [ G2: Heap Buffer Overflow ]
      (Búfer ubicado en el *stack*)              (Búfer asignado en el *heap*)
                      \                                /
                       \                              /
                        +--------------+---------------+
                                       |
                                | COMPUERTA AND |
                                +---------------+
                                /       |       \
                               /        |        \
    [ C1: Búfer de capacidad ]  [ C2: Escritura de ]  [ C3: Sin validación ]
    [     máxima N bytes     ]  [ tamaño M > N     ]  [ previa de límites  ]
                                        |
                                 | COMPUERTA OR |
                                 +--------------+
                                 /      |       \
       +------------------------+       |        +------------------------+
       |                                |                                 |
[ BE1: Entrada externa ]       [ BE2: Cálculo erróneo ]         [ BE3: Copia/lectura ]
[ sin límite verificado]       [ de tamaño o índice   ]         [ sin cota explícita ]
       |                                |                                 |
       +------------------------+       |        +------------------------+
                                        |
                             [ BE4: Manejo deficiente ]
                             [ de punteros o rangos   ]
- **TE** se activa mediante una compuerta **OR** a partir de:
  - **G1 (Stack Buffer Overflow):** desbordamiento sobre un búfer ubicado en la pila (*stack*).
  - **G2 (Heap Buffer Overflow):** desbordamiento sobre un bloque asignado dinámicamente en el *heap*.

### Condiciones típicas (compuerta AND para escritura fuera de rango)
1. Existe un búfer con capacidad máxima $N$.
2. Se realiza una operación de escritura o copia de tamaño $M > N$.
3. No se aplica una validación efectiva de límites antes de escribir.

### Eventos básicos
- **BE1:** Entrada externa o de usuario sin límite de longitud verificado.
- **BE2:** Cálculo erróneo de tamaño o índice (desbordamiento aritmético o conteo incorrecto de bytes/elementos).
- **BE3:** Uso de operaciones o funciones de copia/lectura sin cota explícita de tamaño.
- **BE4:** Manejo deficiente de punteros o rangos en estructuras contiguas.

## 3. Impacto Técnico
- Sobrescritura de variables contiguas, punteros o registros de control.
- Fallos abruptos del programa (*segmentation fault* / comportamiento indefinido).
- Riesgo de integridad y seguridad si datos no confiables alteran el estado del programa.

## 4. Contramedidas y Mitigación
- Validar siempre el tamaño disponible del búfer antes de copiar o escribir.
- Emplear funciones acotadas por longitud y revisar índices y tamaños con cuidado.
- Activar advertencias estrictas en compilación y probar entradas en el límite de capacidad.