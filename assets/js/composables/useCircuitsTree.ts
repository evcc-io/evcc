import type { Circuit, UiLoadpoint } from "@/types/evcc";

export interface CircuitNode {
	name: string;
	circuit: Circuit;
	children: CircuitNode[];
	loadpoints: UiLoadpoint[];
}

export interface CircuitsTree {
	roots: CircuitNode[];
	byName: Record<string, CircuitNode>;
	ungroupedLoadpoints: UiLoadpoint[];
}

export const buildCircuitsTree = (
	circuits: Record<string, Circuit> | undefined,
	loadpoints: UiLoadpoint[] | undefined
): CircuitsTree => {
	const result: CircuitsTree = {
		roots: [],
		byName: {},
		ungroupedLoadpoints: [],
	};

	if (!circuits || Object.keys(circuits).length === 0) {
		result.ungroupedLoadpoints = loadpoints ? [...loadpoints] : [];
		return result;
	}

	const lps = loadpoints ?? [];

	// Create nodes for all circuits
	for (const [name, circuit] of Object.entries(circuits)) {
		result.byName[name] = {
			name,
			circuit,
			children: [],
			loadpoints: [],
		};
	}

	// Wire up parent/child relationships
	for (const node of Object.values(result.byName)) {
		const parentName = node.circuit.parent;
		if (!parentName) {
			result.roots.push(node);
			continue;
		}

		const parentNode = result.byName[parentName];
		if (!parentNode) {
			// parent not found in map, treat as root
			result.roots.push(node);
			continue;
		}

		parentNode.children.push(node);
	}

	// Attach loadpoints to their circuits or keep ungrouped
	for (const lp of lps) {
		const circuitName = lp.circuit;
		if (!circuitName) {
			result.ungroupedLoadpoints.push(lp);
			continue;
		}

		const node = result.byName[circuitName];
		if (!node) {
			result.ungroupedLoadpoints.push(lp);
			continue;
		}

		node.loadpoints.push(lp);
	}

	return result;
};

