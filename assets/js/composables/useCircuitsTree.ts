import type { Circuit, UiLoadpoint } from "@/types/evcc";

export interface CircuitNode {
	name: string;
	circuit: Circuit;
	children: CircuitNode[];
	loadpoints: UiLoadpoint[];
}

export interface CircuitWithMetrics {
	node: CircuitNode;
	level: number;
	power: number;
	maxPower: number;
	hasLimit: boolean;
	usagePercent: number;
	loadpoints: UiLoadpoint[];
}

export interface CircuitsTree {
	roots: CircuitNode[];
	byName: Record<string, CircuitNode>;
	ungroupedLoadpoints: UiLoadpoint[];
	flat: CircuitWithMetrics[];
}

const humanizeCircuitName = (name: string): string =>
	name
		.replace(/[_-]+/g, " ")
		.replace(/\b\w/g, (c) => c.toUpperCase());

export const resolveCircuitTitle = (
	circuit: Circuit | undefined,
	name: string
): string => {
	const title =
		(circuit?.config as { title?: string } | undefined)?.title ??
		circuit?.title;
	if (title) {
		return title;
	}
	return humanizeCircuitName(name);
};

export const buildCircuitsTree = (
	circuits: Record<string, Circuit> | undefined,
	loadpoints: UiLoadpoint[] | undefined
): CircuitsTree => {
	const result: CircuitsTree = {
		roots: [],
		byName: {},
		ungroupedLoadpoints: [],
		flat: [],
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

	const flat: CircuitWithMetrics[] = [];

	const attachMetrics = (circuit: Circuit) => {
		const maxPower =
			circuit.maxPower ?? (circuit.config as any)?.maxPower ?? 0;
		const power = circuit.power ?? 0;
		const hasLimit = maxPower > 0;
		const usagePercent = hasLimit
			? Math.min(100, Math.round((power / maxPower) * 100))
			: 0;
		return { power, maxPower, hasLimit, usagePercent };
	};

	const dfs = (node: CircuitNode, level: number) => {
		const metrics = attachMetrics(node.circuit);
		flat.push({
			node,
			level,
			power: metrics.power,
			maxPower: metrics.maxPower,
			hasLimit: metrics.hasLimit,
			usagePercent: metrics.usagePercent,
			loadpoints: node.loadpoints,
		});
		for (const child of node.children) {
			dfs(child, level + 1);
		}
	};

	for (const root of result.roots) {
		dfs(root, 0);
	}

	// If there are circuits but no tree roots (no parents), fall back to flat list
	if (!flat.length && circuits) {
		for (const [name, circuit] of Object.entries(circuits)) {
			const metrics = attachMetrics(circuit);
			const lps = (loadpoints ?? []).filter((lp) => lp.circuit === name);
			flat.push({
				node: { name, circuit, children: [], loadpoints: lps },
				level: 0,
				power: metrics.power,
				maxPower: metrics.maxPower,
				hasLimit: metrics.hasLimit,
				usagePercent: metrics.usagePercent,
				loadpoints: lps,
			});
		}
	}

	result.flat = flat;

	return result;
};

