import type { Circuit } from "../types/evcc";
import deepClone from "./deepClone";

export interface CircuitNode extends Circuit {
  name?: string;
  children?: CircuitNode[];
}

// circuitTree builds a tree from published circuit data.
// Returns the root node or undefined if empty.
export function circuitTree(circuits?: Record<string, CircuitNode>): CircuitNode | undefined {
  if (circuits === undefined) return;

  let nodes = deepClone(circuits);
  let root: CircuitNode | undefined;

  Object.entries(nodes).forEach(([name, node]) => {
    node.name = name;
    const parent = node.parent ? nodes[node.parent] : undefined;
    if (parent) {
      parent.children ??= [];
      parent.children.push(node);
    } else {
      // found the root
      root = node;
    }
  });

  return root;
}
