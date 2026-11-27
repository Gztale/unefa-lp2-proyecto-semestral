# Comparativa Web Vitals y Presupuesto de Frame: React 19 (VDOM) vs. Svelte 5 (Runes)

## 1. Presupuesto de tiempo por frame (Frame Budget)

Para mantener una interfaz fluida sin tirones ante ráfagas de telemetría 5G, el tiempo total de cómputo y actualización por cuadro debe mantenerse por debajo del presupuesto de cada tasa de refresco:

* **60 Hz:** `1000 ms / 60 = 16.67 ms` por frame.
* **120 Hz:** `1000 ms / 120 = 8.33 ms` por frame.

El tiempo de ráfaga se mide con `performance.now()` alrededor de la generación y recorte de la ventana deslizante de **20 paquetes** (`WINDOW_SIZE = 20`) al inyectar ráfagas de **25 paquetes**.

## 2. React 19 (VDOM) vs. Svelte 5 (Runes)

| Aspecto | React 19 (`TelemetryPanelClient.tsx`) | Svelte 5 (`+page.svelte`) |
| :--- | :--- | :--- |
| **Estado reactivo** | `useState` para `packets` y `burstMs` | `$state` para `packets` y `burstMs` |
| **Valores derivados** | `useMemo` recalculando promedio de latencia y celdas críticas (`> 50 ms`) al cambiar `packets` | `$derived.by` recalculando promedio de latencia y celdas críticas (`> 50 ms`) al cambiar `packets` |
| **Actualización de UI** | Reconciliación de árbol Virtual DOM y actualización de nodos en el DOM real | Actualizaciones quirúrgicas de las expresiones vinculadas al estado reactivo |
| **Ventana deslizante** | `[...prev, ...incoming].slice(-20)` manteniendo los últimos 20 paquetes | `[...packets, ...incoming].slice(-20)` manteniendo los últimos 20 paquetes |
| **Complejidad de actualización** | $O(N)$ (reconciliación/diffing de Virtual DOM) | $O(1)$ (actualización directa vía DAG de señales / Runes) |
| **Tamaño del bundle JS** | Mayor runtime base en cliente (React + reconciliador) | Menor overhead de runtime (compilación AOT más ligera) |
| **Tiempo de hidratación** | Requiere hidratar árbol de componentes en cliente | Menor trabajo de hidratación/activación reactiva |
| **INP (Interaction to Next Paint)** | Sensible a recomputaciones/re-renders en ráfagas | Respuesta más directa al actualizar solo nodos afectados |

## 3. Impacto en Web Vitals y defensa oral

* **INP (Interaction to Next Paint):** Al pulsar el botón de inyectar ráfaga, mantener el cálculo acotado a una ventana fija de 20 elementos evita que el hilo principal se bloquee más allá de `16.67 ms` (60 Hz) o `8.33 ms` (120 Hz).
* **CLS (Cumulative Layout Shift):** Usar una tabla de tamaño acotado por la ventana de 20 paquetes y etiquetas de texto estables evita saltos bruscos de diseño al actualizar métricas.
* **TBT / Long Tasks:** Evitar bucles pesados sobre historiales ilimitados impide tareas largas que retrasen la respuesta a clics o entradas del usuario.