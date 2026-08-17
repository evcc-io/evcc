import type { ConfigCircuit, Circuit } from "../types/evcc";
import deepClone from "./deepClone";

export type CircuitWithName = Circuit & { name: string };
type CircuitInput = ConfigCircuit | CircuitWithName;

export type CircuitNode<T extends CircuitInput> = T & {
  children?: CircuitNode<T>[];
};

function isConfigCircuit(circuit: CircuitInput): circuit is ConfigCircuit {
  return "config" in circuit;
}

function getParent(circuit: CircuitInput): string | undefined {
  if (isConfigCircuit(circuit)) {
    return typeof circuit.config.parent === "string" ? circuit.config.parent : undefined;
  }

  return circuit.parent;
}

// circuitTree builds a tree from published circuit data.
// Returns the root node or undefined if empty.
export function circuitTree<T extends CircuitInput>(circuits?: T[]): CircuitNode<T> | undefined {
  const nodes = deepClone(circuits ?? []) as CircuitNode<T>[];
  const nodesByName = new Map(nodes.map((node) => [node.name, node]));

  let root: CircuitNode<T> | undefined;

  for (const node of nodes) {
    const parentName = getParent(node);
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
