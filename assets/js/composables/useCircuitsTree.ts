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
};

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

const lpsOrEmpty = (loadpoints?: UiLoadpoint[]): UiLoadpoint[] =>
	loadpoints ?? [];

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

const createCircuitNodes = (
	circuits: Record<string, Circuit>
): Record<string, CircuitNode> => {
	const byName: Record<string, CircuitNode> = {};
	for (const [name, circuit] of Object.entries(circuits)) {
		byName[name] = {
			name,
			circuit,
			children: [],
			loadpoints: [],
		};
	}
	return byName;
};

const wireParents = (byName: Record<string, CircuitNode>): CircuitNode[] => {
	const roots: CircuitNode[] = [];
	for (const node of Object.values(byName)) {
		const parentName = node.circuit.parent;
		if (!parentName) {
			roots.push(node);
			continue;
		}
		const parentNode = byName[parentName];
		if (!parentNode) {
			roots.push(node);
			continue;
		}
		parentNode.children.push(node);
	}
	return roots;
};

const attachLoadpointsToNodes = (
	byName: Record<string, CircuitNode>,
	loadpoints: UiLoadpoint[]
): UiLoadpoint[] => {
	const ungrouped: UiLoadpoint[] = [];
	for (const lp of loadpoints) {
		const circuitName = lp.circuit;
		if (!circuitName) {
			ungrouped.push(lp);
			continue;
		}
		const node = byName[circuitName];
		if (!node) {
			ungrouped.push(lp);
			continue;
		}
		node.loadpoints.push(lp);
	}
	return ungrouped;
};

const buildFlatWithMetrics = (
	circuits: Record<string, Circuit>,
	roots: CircuitNode[],
	loadpoints: UiLoadpoint[]
): CircuitWithMetrics[] => {
	const flat: CircuitWithMetrics[] = [];

	const dfs = (node: CircuitNode, level: number) => {
		const metrics = attachMetrics(node.circuit);
		flat.push({
			node,
			level,
			...metrics,
			loadpoints: node.loadpoints,
		});
		for (const child of node.children) {
			dfs(child, level + 1);
		}
	};

	for (const root of roots) {
		dfs(root, 0);
	}

	if (!flat.length) {
		for (const [name, circuit] of Object.entries(circuits)) {
			const metrics = attachMetrics(circuit);
			const lps = loadpoints.filter((lp) => lp.circuit === name);
			flat.push({
				node: { name, circuit, children: [], loadpoints: lps },
				level: 0,
				...metrics,
				loadpoints: lps,
			});
		}
	}

	return flat;
};

export const buildCircuitsTree = (
	circuits: Record<string, Circuit> | undefined,
	loadpoints: UiLoadpoint[] | undefined
): CircuitsTree => {
	const lps = lpsOrEmpty(loadpoints);

	if (!circuits || Object.keys(circuits).length === 0) {
		return {
			roots: [],
			byName: {},
			ungroupedLoadpoints: [...lps],
			flat: [],
		};
	}

	const byName = createCircuitNodes(circuits);
	const roots = wireParents(byName);
	const ungroupedLoadpoints = attachLoadpointsToNodes(byName, lps);
	const flat = buildFlatWithMetrics(circuits, roots, lps);

	return { roots, byName, ungroupedLoadpoints, flat };
};

