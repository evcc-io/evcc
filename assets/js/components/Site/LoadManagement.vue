<template>
	<div
		class="container container--loadpoint px-0 mb-md-2"
		data-testid="load-management"
	>
		<button
			type="button"
			class="btn-reset d-flex align-items-center justify-content-between w-100 mb-3 mt-0 evcc-default-text load-management-header"
			:id="headerId"
			:aria-expanded="open"
			:aria-controls="collapseId"
			:aria-label="$t('main.loadManagement.toggleExpand')"
			@click="open = !open"
		>
			<h3 class="mb-0 evcc-default-text">{{ $t("main.loadManagement.title") }}</h3>
			<shopicon-regular-arrowdropdown
				class="load-management-chevron flex-shrink-0 ms-2"
				:class="{ 'load-management-chevron--open': open }"
				aria-hidden="true"
			></shopicon-regular-arrowdropdown>
		</button>
		<div
			:id="collapseId"
			:aria-labelledby="headerId"
			class="load-management-content"
			:class="{ 'load-management-content--ready': ready }"
			:style="{ height: contentHeight }"
		>
			<div ref="contentInner" class="d-flex flex-column gap-3">
			<div
				v-for="entry in visibleCircuits"
				:key="entry.name"
				class="load-management-circuit rounded p-3 px-sm-4 mx-2 mx-sm-0"
				:class="{
					'load-management-circuit--child': entry.depth > 0,
					'load-management-circuit--expand-trigger': entry.depth === 0,
				}"
				:style="entry.depth ? { marginLeft: (entry.depth * 1.25 + 0.5) + 'rem' } : undefined"
				:role="entry.depth === 0 ? 'button' : undefined"
				:tabindex="entry.depth === 0 ? 0 : undefined"
				:aria-label="
					entry.depth === 0 ? $t('main.loadManagement.toggleExpand') : undefined
				"
				@click.stop="toggleExpandFromRoot(entry)"
				@keydown.enter.prevent="toggleExpandFromRoot(entry)"
				@keydown.space.prevent="toggleExpandFromRoot(entry)"
			>
				<div class="mb-2">
					<div class="d-flex justify-content-between align-items-center">
						<span class="d-flex flex-nowrap flex-grow-1 me-3 align-items-center text-truncate">
							<shopicon-regular-powersupply class="flex-shrink-0 me-2"></shopicon-regular-powersupply>
							<span class="text-truncate evcc-default-text">
								{{ circuitTitle(entry.circuit, entry.name) }}
							</span>
						</span>
						<span class="text-end text-nowrap ps-1 d-flex align-items-center gap-2 flex-shrink-0">
							<span
								v-if="entry.circuit.dimmed"
								class="badge bg-warning text-dark"
								data-bs-toggle="tooltip"
								:title="$t('main.loadManagement.dimmedTooltip')"
							>
								{{ $t("main.loadManagement.dimmed") }}
							</span>
							<span
								v-if="entry.circuit.curtailed"
								class="badge bg-secondary"
								data-bs-toggle="tooltip"
								:title="$t('main.loadManagement.curtailedTooltip')"
							>
								{{ $t("main.loadManagement.curtailed") }}
							</span>
							<span
								v-if="showHeaderPower(entry.circuit)"
								class="fw-bold evcc-default-text"
							>
								<template v-if="hasLimit(entry.circuit)">
									<AnimatedNumber
										:to="entry.circuit.power ?? 0"
										:format="kwFormat"
									/>
									/
									<AnimatedNumber
										:to="entry.circuit.maxPower"
										:format="kwFormat"
									/>
								</template>
								<template v-else>
									<AnimatedNumber
										:to="entry.circuit.power ?? 0"
										:format="kwFormat"
									/>
								</template>
							</span>
						</span>
					</div>
				</div>
				<!-- Power bar: 0 ——————— max power (kW) -->
				<div v-if="hasLimit(entry.circuit)" class="load-management-bars mt-2">
					<div class="d-flex justify-content-between small evcc-gray mb-1">
						<span>{{ $t("main.loadManagement.power") }}</span>
						<span class="text-nowrap">
							0 —
							<AnimatedNumber
								:to="entry.circuit.maxPower"
								:format="kwFormat"
							/>
						</span>
					</div>
					<div
						class="load-management-progress progress"
						role="progressbar"
						:aria-valuenow="entry.circuit.power ?? 0"
						aria-valuemin="0"
						:aria-valuemax="entry.circuit.maxPower"
					>
						<div
							class="progress-bar load-management-bar-fill"
							:class="{ 'load-management-bar-fill--warning': (entry.circuit.power ?? 0) >= entry.circuit.maxPower }"
							:style="{ width: usagePercent(entry.circuit) + '%', ...transition }"
						>
							<span
								v-if="(entry.circuit.power ?? 0) > 0"
								class="progress-bar-value"
							>
								<AnimatedNumber
									:to="entry.circuit.power ?? 0"
									:format="kwFormat"
								/>
							</span>
						</div>
					</div>
				</div>
				<!-- Current bar: 0 ——————— max current (A) -->
				<div v-if="hasCurrentInfo(entry.circuit)" class="load-management-bars mt-3">
					<div class="d-flex justify-content-between small evcc-gray mb-1">
						<span>{{ $t("main.loadManagement.current") }}</span>
						<span class="text-nowrap">
							0 —
							<AnimatedNumber
								:to="entry.circuit.maxCurrent"
								:format="maxCurrentFormat"
							/>
							A
						</span>
					</div>
					<div
						class="load-management-progress progress"
						role="progressbar"
						:aria-valuenow="entry.circuit.current ?? 0"
						aria-valuemin="0"
						:aria-valuemax="entry.circuit.maxCurrent"
					>
						<div
							class="progress-bar load-management-bar-fill"
							:class="{ 'load-management-bar-fill--warning': (entry.circuit.current ?? 0) >= entry.circuit.maxCurrent }"
							:style="{ width: currentPercent(entry.circuit) + '%', ...transition }"
						>
							<span
								v-if="(entry.circuit.current ?? 0) > 0"
								class="progress-bar-value"
							>
								<AnimatedNumber
									:to="entry.circuit.current ?? 0"
									:format="(v) => maxCurrentFormat(v) + ' A'"
								/>
							</span>
						</div>
					</div>
				</div>
			</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/arrowdropdown";
import { defineComponent, type PropType, computed, getCurrentInstance } from "vue";
import type { Circuit } from "@/types/evcc";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import AnimatedNumber from "../Helper/AnimatedNumber.vue";
import { resolveCircuitTitle } from "@/composables/useCircuitsTree";

export default defineComponent({
	name: "LoadManagement",
	components: { AnimatedNumber },
	mixins: [formatter],
	props: {
		circuits: {
			type: Object as PropType<Record<string, Circuit>>,
			default: () => ({}),
		},
	},
	data() {
		return {
			open: false,
			ready: false,
			contentCompleteHeight: null as number | null,
			collapsedHeight: null as number | null,
		};
	},
	setup(props: { circuits?: Record<string, Circuit> }) {
		const circuitsList = computed(() => props.circuits ?? {});
		const transition = { transition: "width var(--evcc-transition-fast) linear" };
		const instance = getCurrentInstance();
		const idSuffix = instance?.uid ?? Math.random().toString(36).slice(2, 9);
		const collapseId = `load-management-content-${idSuffix}`;
		const headerId = `load-management-header-${idSuffix}`;
		const orderedCircuits = computed(() => {
			const circuits = circuitsList.value;
			const entries: { name: string; circuit: Circuit; depth: number }[] = [];
			const byParent = new Map<string, string[]>();
			const names = Object.keys(circuits).sort((a, b) => a.localeCompare(b));
			for (const name of names) {
				const c = circuits[name];
				if (c === undefined) continue;
				const parent = c.parent ?? "";
				if (!byParent.has(parent)) byParent.set(parent, []);
				byParent.get(parent)!.push(name);
			}
			const pushChildren = (parentKey: string, depth: number) => {
				const children = byParent.get(parentKey);
				if (!children) return;
				for (const name of children) {
					const circuit = circuits[name];
					if (circuit === undefined) continue;
					entries.push({ name, circuit, depth });
					pushChildren(name, depth + 1);
				}
			};
			pushChildren("", 0);
			return entries;
		});
		return { circuitsList, orderedCircuits, transition, collapseId, headerId };
	},
	computed: {
		rootCircuits(): { name: string; circuit: Circuit; depth: number }[] {
			return this.orderedCircuits.filter((e) => e.depth === 0);
		},
		visibleCircuits(): { name: string; circuit: Circuit; depth: number }[] {
			return this.open ? this.orderedCircuits : this.rootCircuits;
		},
		contentHeight(): string {
			if (this.open && this.contentCompleteHeight != null) {
				return `${this.contentCompleteHeight}px`;
			}
			if (!this.open) {
				const h = this.collapsedHeight ?? this.contentCompleteHeight;
				if (h != null) return `${h}px`;
			}
			return "0";
		},
	},
	watch: {
		open() {
			this.$nextTick(this.updateHeight);
		},
		orderedCircuits: {
			handler() {
				this.$nextTick(this.updateHeight);
			},
			deep: true,
		},
	},
	mounted() {
		this.ready = true;
		const el = this.$refs["contentInner"] as HTMLElement | undefined;
		if (el) {
			this.updateHeight();
			const ro = new ResizeObserver(() => this.updateHeight());
			ro.observe(el);
			(this as { _resizeObserver?: ResizeObserver })._resizeObserver = ro;
		}
	},
	unmounted() {
		(this as { _resizeObserver?: ResizeObserver })._resizeObserver?.disconnect();
	},
	methods: {
		toggleExpandFromRoot(entry: { depth: number }) {
			if (entry.depth === 0) this.open = !this.open;
		},
		updateHeight() {
			const height =
				(this.$refs["contentInner"] as HTMLElement | undefined)?.offsetHeight ?? 0;
			if (this.open) {
				this.contentCompleteHeight = height;
			} else {
				this.collapsedHeight = height;
			}
		},
		showHeaderPower(circuit: Circuit): boolean {
			return this.hasLimit(circuit) || typeof circuit.power === "number";
		},
		kwFormat(v: number): string {
			return this.fmtW(v, POWER_UNIT.KW);
		},
		circuitTitle(circuit: Circuit, name: string): string {
			return resolveCircuitTitle(circuit, name);
		},
		hasLimit(circuit: Circuit): boolean {
			return typeof circuit.maxPower === "number" && circuit.maxPower > 0;
		},
		hasCurrentInfo(circuit: Circuit): boolean {
			return (
				typeof circuit.maxCurrent === "number" &&
				circuit.maxCurrent > 0 &&
				circuit.current != null
			);
		},
		usagePercent(circuit: Circuit): number {
			if (!this.hasLimit(circuit)) return 0;
			const power = circuit.power ?? 0;
			const max = circuit.maxPower!;
			return Math.min(100, Math.round((power / max) * 100));
		},
		currentPercent(circuit: Circuit): number {
			if (!this.hasCurrentInfo(circuit)) return 0;
			const current = circuit.current ?? 0;
			const max = circuit.maxCurrent!;
			return Math.min(100, Math.round((current / max) * 100));
		},
		maxCurrentFormat(v: number): string {
			return Number(v.toFixed(1)).toString();
		},
	},
});
</script>

<style scoped>
.load-management-header {
	cursor: pointer;
	text-align: left;
}

.load-management-header:hover {
	opacity: 0.9;
}

.load-management-chevron {
	transition: transform var(--evcc-transition-medium) ease;
	transform: rotate(-90deg);
}

.load-management-chevron--open {
	transform: rotate(0deg);
}

.load-management-content {
	overflow: hidden;
	transition-property: height;
	transition-duration: 0;
	transition-timing-function: ease-out;
}

.load-management-content--ready {
	transition-duration: var(--evcc-transition-medium);
}

.load-management-circuit {
	background-color: var(--evcc-box);
}

.load-management-circuit--expand-trigger {
	cursor: pointer;
}

.load-management-circuit--expand-trigger:hover {
	opacity: 0.95;
}

.load-management-circuit--child {
	border-left: 6px solid var(--evcc-gray);
	box-shadow: inset 10px 0 14px -6px rgba(0, 0, 0, 0.12);
}

.load-management-progress {
	height: 32px;
	font-size: 1rem;
	background: var(--evcc-background);
	position: relative;
}

.load-management-progress .progress-bar {
	position: relative;
	min-width: 0;
	display: flex;
	align-items: center;
}

.load-management-bar-fill {
	background-color: var(--evcc-dark-green);
}

.load-management-bar-fill--warning {
	background-color: var(--evcc-orange);
}

.progress-bar-value {
	position: absolute;
	left: 0.5rem;
	top: 50%;
	transform: translateY(-50%);
	font-size: 0.875rem;
	font-weight: 600;
	color: var(--evcc-default-text);
	white-space: nowrap;
	text-shadow: 0 0 2px var(--evcc-box);
	pointer-events: none;
}
</style>
