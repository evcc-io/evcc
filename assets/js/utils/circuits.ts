import type { Circuit } from "../types/evcc";

export interface CircuitNode extends Circuit {
  name: string;
  children?: CircuitNode[];
}

// circuitTree builds a tree from published circuit data.
// Returns the root node or null if empty.
export function circuitTree(circuits: Record<string, Circuit>): CircuitNode | null {
  const nodes = new Map<string, CircuitNode>();
  for (const [name, circuit] of Object.entries(circuits)) {
    nodes.set(name, { ...circuit, name });
  }

  let root: CircuitNode | null = null;
  for (const [name, circuit] of Object.entries(circuits)) {
    if (circuit.parent && nodes.has(circuit.parent)) {
      const parent = nodes.get(circuit.parent)!;
      parent.children = parent.children || [];
      parent.children.push(nodes.get(name)!);
    } else {
      root = nodes.get(name)!;
    }
  }

  return root;
}
