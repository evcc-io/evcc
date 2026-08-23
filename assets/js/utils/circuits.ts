import type { ConfigCircuit, Circuit } from "../types/evcc";
import deepClone from "./deepClone";

export type ConfigCircuitNode = ConfigCircuit & {
  children?: ConfigCircuitNode[];
};

export type CircuitNode = Circuit & {
  children?: CircuitNode[];
};

// configCircuitTree builds a tree from ConfigCircuit data
export function configCircuitTree(circuits?: ConfigCircuit[]): ConfigCircuitNode | undefined {
  const nodes = deepClone(circuits ?? []) as ConfigCircuitNode[];
  const nodesByName = new Map(nodes.map((node) => [node.name, node]));

  let root: ConfigCircuitNode | undefined;

  for (const node of nodes) {
    const parentName = typeof node.config.parent === "string" ? node.config.parent : undefined;
    const parent = parentName ? nodesByName.get(parentName) : undefined;

    if (parent) {
      parent.children ??= [];
      parent.children.push(node);
    } else {
      root = node;
    }
  }

  return root;
}

// circuitTree builds a tree from published Circuit data (Record keyed by id)
export function circuitTree(circuits?: Record<string, Circuit>): CircuitNode | undefined {
  const source = deepClone(circuits ?? {}) as Record<string, CircuitNode>;
  const entries = Object.entries(source);

  const nodeById = new Map(entries);

  let root: CircuitNode | undefined;

  for (const [, node] of entries) {
    const parent = node.parent ? nodeById.get(node.parent) : undefined;

    if (parent) {
      parent.children ??= [];
      parent.children.push(node);
    } else {
      root = node;
    }
  }

  return root;
}
