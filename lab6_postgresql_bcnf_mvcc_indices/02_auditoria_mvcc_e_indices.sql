BEGIN;

INSERT INTO catalogo_planes_bcnf (plan_codigo, nombre_plan, tarifa_centavos, moneda, activo)
VALUES
    ('PLAN-5G-100', '5G Hogar 100 Mbps', 2500, 'USD', TRUE),
    ('PLAN-5G-300', '5G Hogar 300 Mbps', 4000, 'USD', TRUE),
    ('PLAN-5G-MOV', '5G Móvil Ilimitado', 1500, 'USD', TRUE)
ON CONFLICT (plan_codigo) DO NOTHING;

-- Lote de 50.000 abonados con generate_series
INSERT INTO abonados_bcnf (msisdn, documento_ident, nombre_completo, plan_codigo, saldo_centavos, estado_cuenta)
SELECT
    '58412' || LPAD(g::text, 7, '0'),
    'V-' || (20000000 + g)::text,
    'Abonado Prueba ' || g::text,
    CASE (g % 3)
        WHEN 0 THEN 'PLAN-5G-100'
        WHEN 1 THEN 'PLAN-5G-300'
        ELSE 'PLAN-5G-MOV'
    END,
    10000,
    'ACTIVO'
FROM generate_series(1, 50000) AS g
ON CONFLICT (msisdn) DO NOTHING;

-- Lote de 50.000 transacciones con metadata_jsonb variado y casos objetivo
INSERT INTO transacciones_bss (abonado_id, idempotency_key, tipo_operacion, monto_centavos, estado_tx, metadata_jsonb, created_at)
SELECT
    a.abonado_id,
    'IDEMP-LAB6-' || g::text,
    'CONSUMO_DATOS_5G',
    500 + (g % 2000),
    'CONFIRMADA',
    CASE
        WHEN g % 250 = 0 THEN '{"antena_5g": "CCS-NORTE-01", "roaming": true, "celda": "NR-01"}'::jsonb
        ELSE jsonb_build_object(
            'antena_5g', 'CCS-SUR-' || LPAD(((g % 20) + 1)::text, 2, '0'),
            'roaming', (g % 7 = 0),
            'celda', 'NR-' || LPAD(((g % 50) + 1)::text, 2, '0')
        )
    END,
    NOW() - ((50000 - g) || ' seconds')::interval
FROM generate_series(1, 50000) AS g
JOIN abonados_bcnf a ON a.msisdn = '58412' || LPAD(g::text, 7, '0')
ON CONFLICT (idempotency_key) DO NOTHING;

INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload_jsonb, status)
SELECT
    'ABONADO',
    a.msisdn,
    'SALDO_ACTUALIZADO',
    jsonb_build_object('msisdn', a.msisdn, 'saldo_centavos', a.saldo_centavos),
    'PENDING'
FROM abonados_bcnf a
WHERE a.msisdn IN ('584120000001', '584120000002')
  AND NOT EXISTS (
      SELECT 1 FROM outbox_events o WHERE o.aggregate_id = a.msisdn AND o.status = 'PENDING'
  );

COMMIT;

-- 1. Inspección MVCC: UPDATE sobre abonado y consulta de ctid, xmin, xmax, saldo_centavos
UPDATE abonados_bcnf
SET saldo_centavos = saldo_centavos - 1500
WHERE msisdn = '584120000001';

SELECT ctid, xmin, xmax, msisdn, saldo_centavos
FROM abonados_bcnf
WHERE msisdn = '584120000001';

ANALYZE abonados_bcnf;
ANALYZE transacciones_bss;
ANALYZE outbox_events;

-- 2. Plan ANTES de crear los índices GIN y BRIN (esperable Seq Scan)
EXPLAIN (ANALYZE, BUFFERS)
SELECT transaccion_id, abonado_id, monto_centavos, metadata_jsonb
FROM transacciones_bss
WHERE metadata_jsonb @> '{"antena_5g": "CCS-NORTE-01", "roaming": true}';

-- 3. Creación de índices GIN (jsonb_path_ops), BRIN (created_at) y B-Tree/Parcial
CREATE INDEX IF NOT EXISTS idx_tx_metadata_gin
    ON transacciones_bss USING GIN (metadata_jsonb jsonb_path_ops);

CREATE INDEX IF NOT EXISTS idx_tx_created_at_brin
    ON transacciones_bss USING BRIN (created_at);

CREATE INDEX IF NOT EXISTS idx_tx_abonado_created
    ON transacciones_bss (abonado_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_outbox_pending_created
    ON outbox_events (created_at)
    WHERE status = 'PENDING';

ANALYZE transacciones_bss;
ANALYZE outbox_events;

-- 4. Plan DESPUÉS de crear los índices (esperable Bitmap Index Scan en JSONB)
EXPLAIN (ANALYZE, BUFFERS)
SELECT transaccion_id, abonado_id, monto_centavos, metadata_jsonb
FROM transacciones_bss
WHERE metadata_jsonb @> '{"antena_5g": "CCS-NORTE-01", "roaming": true}';
