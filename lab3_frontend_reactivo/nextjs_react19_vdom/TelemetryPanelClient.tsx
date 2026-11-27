// @ts-nocheck
"use client";

import React, { useMemo, useState } from "react";

export interface TelemetryPacket {
  cellId: string;
  latencyMs: number;
  packetLossPct: number;
  throughputMbps: number;
}

const WINDOW_SIZE = 20;

const randomPacket = (): TelemetryPacket => ({
  cellId: `CELL-${Math.floor(Math.random() * 5) + 1}`,
  latencyMs: Number((10 + Math.random() * 80).toFixed(1)),
  packetLossPct: Number((Math.random() * 3).toFixed(2)),
  throughputMbps: Number((100 + Math.random() * 900).toFixed(1)),
});

export default function TelemetryPanelClient() {
  const [packets, setPackets] = useState<TelemetryPacket[]>(() =>
    Array.from({ length: WINDOW_SIZE }, randomPacket)
  );
  const [burstMs, setBurstMs] = useState<number>(0);

  const stats = useMemo(() => {
    if (packets.length === 0) return { avgLatency: 0, criticalCells: 0 };
    const sumLatency = packets.reduce((acc, p) => acc + p.latencyMs, 0);
    const criticalCells = new Set(
      packets.filter((p) => p.latencyMs > 50).map((p) => p.cellId)
    ).size;

    return {
      avgLatency: Number((sumLatency / packets.length).toFixed(2)),
      criticalCells,
    };
  }, [packets]);

  const injectBurst = (count = 25) => {
    const t0 = performance.now();
    const incoming = Array.from({ length: count }, randomPacket);
    setPackets((prev) => [...prev, ...incoming].slice(-WINDOW_SIZE));
    const t1 = performance.now();
    setBurstMs(Number((t1 - t0).toFixed(4)));
  };

  return (
    <div>
      <button type="button" onClick={() => injectBurst(25)}>
        Inyectar ráfaga
      </button>
      <p>Latencia promedio: {stats.avgLatency} ms</p>
      <p>Celdas críticas (&gt;50 ms): {stats.criticalCells}</p>
      <p>Tiempo de ráfaga: {burstMs} ms</p>
      <table>
        <thead>
          <tr>
            <th>cellId</th>
            <th>latencyMs</th>
            <th>packetLossPct</th>
            <th>throughputMbps</th>
          </tr>
        </thead>
        <tbody>
          {packets.map((p, i) => (
            <tr key={`${p.cellId}-${i}`}>
              <td>{p.cellId}</td>
              <td>{p.latencyMs}</td>
              <td>{p.packetLossPct}</td>
              <td>{p.throughputMbps}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}