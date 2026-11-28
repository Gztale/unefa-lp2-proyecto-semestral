<<<<<<< Updated upstream
services:
  postgres:
    image: postgres:16-alpine
    container_name: unefa_pg16
    environment:
      POSTGRES_DB: unefa_bss
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres_password
    ports:
      - "5432:5432"
    volumes:
      - pg16_data:/var/lib/postgresql/data
    deploy:
      resources:
        limits:
          memory: 1g
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d unefa_bss"]
      interval: 5s
      timeout: 5s
      retries: 10
    restart: unless-stopped

  scylla-node1:
    image: scylladb/scylla:5.4
    container_name: scylla-node1
    command: --smp 1 --memory 1G --overprovisioned 1 --developer-mode 1
    ports:
      - "9042:9042"
    volumes:
      - scylla_data:/var/lib/scylla
    deploy:
      resources:
        limits:
          memory: 1536m
    healthcheck:
      test: ["CMD-SHELL", "cqlsh -e 'SELECT now() FROM system.local;'"]
      interval: 10s
      timeout: 10s
      retries: 15
    restart: unless-stopped

volumes:
  pg16_data:
  scylla_data:
=======
BEGIN;

-- Catálogo de planes normalizado en BCNF (sin dependencias transitivas)
CREATE TABLE IF NOT EXISTS catalogo_planes_bcnf (
    plan_codigo      VARCHAR(32) PRIMARY KEY,
    nombre_plan      VARCHAR(100) NOT NULL,
    tarifa_centavos  BIGINT NOT NULL CHECK (tarifa_centavos >= 0),
    moneda           CHAR(3) NOT NULL DEFAULT 'USD',
    activo           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Abonados con saldo en centavos (evitando flotantes para dinero)
CREATE TABLE IF NOT EXISTS abonados_bcnf (
    abonado_id       BIGSERIAL PRIMARY KEY,
    msisdn           VARCHAR(20) NOT NULL UNIQUE,
    documento_ident  VARCHAR(32) NOT NULL UNIQUE,
    nombre_completo  VARCHAR(120) NOT NULL,
    plan_codigo      VARCHAR(32) NOT NULL REFERENCES catalogo_planes_bcnf(plan_codigo) ON DELETE RESTRICT,
    saldo_centavos   BIGINT NOT NULL DEFAULT 0 CHECK (saldo_centavos >= 0),
    estado_cuenta    VARCHAR(20) NOT NULL DEFAULT 'ACTIVO'
        CHECK (estado_cuenta IN ('ACTIVO', 'SUSPENDIDO', 'BAJA')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Transacciones BSS con metadatos semiestructurados JSONB
CREATE TABLE IF NOT EXISTS transacciones_bss (
    transaccion_id   BIGSERIAL PRIMARY KEY,
    abonado_id       BIGINT NOT NULL REFERENCES abonados_bcnf(abonado_id) ON DELETE RESTRICT,
    idempotency_key  VARCHAR(100) NOT NULL UNIQUE,
    tipo_operacion   VARCHAR(40) NOT NULL,
    monto_centavos   BIGINT NOT NULL CHECK (monto_centavos >= 0),
    estado_tx        VARCHAR(20) NOT NULL DEFAULT 'CONFIRMADA'
        CHECK (estado_tx IN ('PENDIENTE', 'CONFIRMADA', 'REVERSADA')),
    metadata_jsonb   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tabla Transactional Outbox para sincronización asíncrona sin dual-write
CREATE TABLE IF NOT EXISTS outbox_events (
    event_id         BIGSERIAL PRIMARY KEY,
    aggregate_type   VARCHAR(50) NOT NULL,
    aggregate_id     VARCHAR(80) NOT NULL,
    event_type       VARCHAR(60) NOT NULL,
    payload_jsonb    JSONB NOT NULL,
    status           VARCHAR(20) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'PROCESSED', 'FAILED')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at     TIMESTAMPTZ
);

COMMIT
>>>>>>> Stashed changes
