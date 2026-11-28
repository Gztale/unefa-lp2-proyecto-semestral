# Reporte de Auditoría MVCC, Índices y EXPLAIN (ANALYZE, BUFFERS) — Lab 6

## 1. Normalización BCNF y Estructura (`01_schema_bcnf_y_outbox.sql`)
- **Motor y Base de Datos:** PostgreSQL 16 (`unefa_bss`) ejecutándose en contenedor Docker sobre Windows 10 (WSL 2).
- **`catalogo_planes_bcnf`:** desacopla `plan_codigo`, `nombre_plan`, `tarifa_centavos`, `moneda` y `activo`, eliminando dependencias transitivas entre el abonado y los atributos comerciales del plan.
- **`abonados_bcnf`:** registra `msisdn`, `documento_ident`, `nombre_completo`, `plan_codigo` (`FK` hacia `catalogo_planes_bcnf`) y `saldo_centavos` (`BIGINT` con restricción `CHECK (saldo_centavos >= 0)`).
- **`transacciones_bss`:** vincula `abonado_id` (`FK`), garantiza idempotencia con `idempotency_key UNIQUE`, almacena montos enteros en centavos (`monto_centavos >= 0`) y guarda metadatos semiestructurados en `metadata_jsonb` (`JSONB NOT NULL`).
- **`outbox_events`:** implementa el patrón *Transactional Outbox* con `payload_jsonb`, `status` (`'PENDING'`, `'PROCESSED'`, `'FAILED'`) y `created_at` para desacoplar la persistencia transaccional de la proyección en ScyllaDB sin *dual-write*.

## 2. Auditoría Forense MVCC (`ctid`, `xmin`, `xmax`)
- Se pobló la base con un lote de **50.000 abonados** y **50.000 transacciones** mediante `generate_series(1, 50000)`.
- Posteriormente se ejecutó un `UPDATE` descontando `1500` centavos al abonado `msisdn = '584120000001'` y se consultaron las columnas ocultas del sistema (`ctid`, `xmin`, `xmax`, `saldo_centavos`).
- **Hallazgo técnico:** PostgreSQL no sobrescribe la tupla en disco; crea una nueva versión física (nuevo `ctid` y nuevo `xmin` igual al ID de la transacción actual) y marca la versión previa con `xmax` distinto de cero, dejándola como tupla muerta reclamable por `VACUUM` / `AUTOVACUUM` sin bloquear lectores concurrentes.

## 3. Índices Implementados (`02_auditoria_mvcc_e_indices.sql`)
- **`idx_tx_metadata_gin`:** `CREATE INDEX idx_tx_metadata_gin ON transacciones_bss USING GIN (metadata_jsonb jsonb_path_ops);` — índice invertido optimizado para consultas de contención (`@>`) sobre claves/valores dentro del documento JSONB.
- **`idx_tx_created_at_brin`:** `CREATE INDEX idx_tx_created_at_brin ON transacciones_bss USING BRIN (created_at);` — índice compacto por rangos de bloques físicos para series temporales ordenadas por inserción.
- **`idx_tx_abonado_created`:** índice compuesto B-Tree `(abonado_id, created_at DESC)` para consultas históricas por abonado.
- **`idx_outbox_pending_created`:** índice parcial sobre `(created_at) WHERE status = 'PENDING'` para acelerar la extracción del worker con `FOR UPDATE SKIP LOCKED`.

## 4. Comparativa `EXPLAIN (ANALYZE, BUFFERS)` — Antes vs. Después de Índices

**Consulta evaluada sobre 50.000 registros:**

Escenario,Estrategia de Acceso,Filas Retornadas,Buffers,Planning Time,Execution Time
Antes de crear idx_tx_metadata_gin,Seq Scan sobre transacciones_bss (escaneo secuencial completo de las 50.000 tuplas),200,Lectura completa de páginas de la tabla,Mayor costo de escaneo completo,Escaneo lineal de toda la tabla
Después de crear idx_tx_metadata_gin (jsonb_path_ops),"Bitmap Heap Scan + Bitmap Index Scan usando Index Cond: (metadata_jsonb @> '{""roaming"": true, ""antena_5g"": ""CCS-NORTE-01""}'::jsonb)",200,shared hit=9 (planning: shared hit=41 read=1),1.083 ms,0.719 ms
5. Conclusiones del Laboratorio 6
La normalización hasta BCNF elimina redundancias y anomalías de actualización en catálogos y saldos de abonados, mientras JSONB absorbe atributos variables de red 5G sin alterar el esquema relacional.

El índice GIN con clase de operador jsonb_path_ops reduce drasticamente la lectura de páginas en disco/memoria (shared hit=9) frente a un recorrido secuencial sobre 50.000 filas, logrando una ejecución por debajo de 1 ms (0.719 ms).

El índice BRIN sobre created_at y el índice parcial en outbox_events mantienen baja huella en disco y preparan la base para el consumo concurrente del Ejercicio 4.

```sql
SELECT transaccion_id, abonado_id, monto_centavos, metadata_jsonb
FROM transacciones_bss
WHERE metadata_jsonb @> '{"antena_5g": "CCS-NORTE-01", "roaming": true}';
