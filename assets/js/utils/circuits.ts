import type { ConfigCircuit, ConfigMeter, Circuit } from "../types/evcc";

export type ConfigCircuitNode = ConfigCircuit & {
  children?: ConfigCircuitNode[];
};

export type CircuitNode = Circuit & {
  children?: CircuitNode[];
};

// buildTree links nodes to their parents and returns the root
function buildTree<T extends { children?: T[] }>(
  byKey: Record<string, T>,
  parentOf: (node: T) => string | undefined
): T | undefined {
  const nodes = Object.fromEntries(Object.entries(byKey).map(([k, v]) => [k, { ...v }]));
  let root: T | undefined;
  for (const node of Object.values(nodes)) {
    const parent = nodes[parentOf(node) ?? ""];
    if (parent) {
      (parent.children ??= []).push(node);
    } else {
      root = node;
    }
  }
  return root;
}

// configCircuitTree builds a tree from ConfigCircuit data
export function configCircuitTree(circuits: ConfigCircuit[] = []): ConfigCircuitNode | undefined {
  return buildTree<ConfigCircuitNode>(
    Object.fromEntries(circuits.map((c) => [c.name, c])),
    (node) => (typeof node.config.parent === "string" ? node.config.parent : undefined)
  );
}

// circuitTree builds a tree from published Circuit data (Record keyed by id)
export function circuitTree(circuits: Record<string, Circuit> = {}): CircuitNode | undefined {
  return buildTree<CircuitNode>(circuits, (node) => node.parent);
}

// meterTitle returns the display name of a referenced meter
export function meterTitle(meters: ConfigMeter[], name?: string): string {
  const meter = meters.find((m) => m.name === name);
  return meter?.deviceProduct || meter?.config?.template || "";
}

export type LimitState = "normal" | "warning" | "danger";

export interface LimitBar {
  state: LimitState;
  /** Bar width up to the limit in %, the excess fills the rest. */
  fill: number;
}

// limitBar scales to the limit, or to the value once it exceeds the limit
export function limitBar(value: number, limit?: number): LimitBar {
  if (!limit) return { state: "normal", fill: 100 };
  const ratio = Math.max(0, value) / limit;
  if (ratio > 1) return { state: "danger", fill: 100 / ratio };
  if (ratio >= 0.95) return { state: "warning", fill: 100 };
  return { state: "normal", fill: ratio * 100 };
}
