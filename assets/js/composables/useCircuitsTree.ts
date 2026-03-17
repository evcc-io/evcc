import type { Circuit, UiLoadpoint } from "@/types/evcc";
import { circuitTree, type CircuitNode as BaseCircuitNode } from "@/utils/circuits";

export interface CircuitNodeWithLoadpoints extends BaseCircuitNode {
	children: CircuitNodeWithLoadpoints[];
	loadpoints: UiLoadpoint[];
}

export interface CircuitWithMetrics {
	node: CircuitNodeWithLoadpoints;
	level: number;
	power: number;
	maxPower: number;
	hasLimit: boolean;
	usagePercent: number;
	loadpoints: UiLoadpoint[];
}

export interface CircuitsTree {
	root: CircuitNodeWithLoadpoints | null;
	flat: CircuitWithMetrics[];
	ungroupedLoadpoints: UiLoadpoint[];
}

const lpsOrEmpty = (loadpoints?: UiLoadpoint[]): UiLoadpoint[] =>
	loadpoints ?? [];

const attachMetrics = (node: CircuitNodeWithLoadpoints) => {
	const maxPower =
		node.maxPower ?? (node.config as any)?.maxPower ?? 0;
	const power = node.power ?? 0;
	const hasLimit = maxPower > 0;
	const usagePercent = hasLimit
		? Math.min(100, Math.round((power / maxPower) * 100))
		: 0;
	return { power, maxPower, hasLimit, usagePercent };
};

const cloneTreeWithLoadpoints = (
	root: BaseCircuitNode | null
): CircuitNodeWithLoadpoints | null => {
	if (!root) return null;
	const clone = (node: BaseCircuitNode): CircuitNodeWithLoadpoints => ({
		...node,
		children: (node.children ?? []).map(clone),
		loadpoints: [],
	});
	return clone(root);
};

const attachLoadpointsToTree = (
	root: CircuitNodeWithLoadpoints | null,
	loadpoints: UiLoadpoint[]
): UiLoadpoint[] => {
	const byName: Record<string, CircuitNodeWithLoadpoints> = {};
	const ungrouped: UiLoadpoint[] = [];

	if (root) {
		const stack: CircuitNodeWithLoadpoints[] = [root];
		while (stack.length) {
			const node = stack.pop()!;
			byName[node.name] = node;
			for (const child of node.children) {
				stack.push(child);
			}
		}
	}

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
	root: CircuitNodeWithLoadpoints | null
): CircuitWithMetrics[] => {
	const flat: CircuitWithMetrics[] = [];

	if (!root) {
		return flat;
	}

	const dfs = (node: CircuitNodeWithLoadpoints, level: number) => {
		const metrics = attachMetrics(node);
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

	dfs(root, 0);

	return flat;
};

export const buildCircuitsTree = (
	circuits: Record<string, Circuit> | undefined,
	loadpoints: UiLoadpoint[] | undefined
): CircuitsTree => {
	const lps = lpsOrEmpty(loadpoints);

	if (!circuits || Object.keys(circuits).length === 0) {
		return {
			root: null,
			flat: [],
			ungroupedLoadpoints: [...lps],
		};
	}

	const rootFromCircuits = circuitTree(circuits);
	const root = cloneTreeWithLoadpoints(rootFromCircuits as any);
	const ungroupedLoadpoints = attachLoadpointsToTree(root, lps);
	const flat = buildFlatWithMetrics(root);

	return { root, flat, ungroupedLoadpoints };
};

