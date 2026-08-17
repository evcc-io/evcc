import type { ConfigCircuit } from "../types/evcc";
import deepClone from "./deepClone";

export interface CircuitNode extends ConfigCircuit {
  children?: CircuitNode[];
}

// circuitTree builds a tree from published circuit data.
// Returns the root node or undefined if empty.
export function circuitTree(circuits?: CircuitNode[]): CircuitNode | undefined {
  const nodes = deepClone(circuits ?? []);
  const nodesByName = new Map(nodes.map((node) => [node.name, node]));

  let root: CircuitNode | undefined;

  for (const node of nodes) {
    const parent = node.config.parent ? nodesByName.get(node.config.parent as string) : undefined;

    if (parent) {
      parent.children ??= [];
      parent.children.push(node);
    } else {
      root = node;
    }
  }

  return root;
}
