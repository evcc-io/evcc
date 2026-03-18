<template>
	<teleport to="body">
		<div v-if="show" class="circuits-modal-backdrop">
			<div
				ref="dialog"
				class="circuits-modal-dialog modal-dialog-scrollable modal-lg modal-fullscreen-md-down"
				role="dialog"
				aria-modal="true"
				aria-labelledby="circuitsModalLabel"
				tabindex="-1"
			>
				<div class="modal-content">
					<div class="modal-header">
						<h5 id="circuitsModalLabel" class="modal-title">
							{{ $t("main.circuits.title") }}
						</h5>
						<button type="button" class="btn-close" aria-label="Close" @click="close"></button>
					</div>
					<div class="modal-body">
						<p class="small text-muted mb-3">
							{{ $t("main.circuits.help") }}
						</p>
						<div v-if="!hasTree">
							<p class="mb-0">
								{{ $t("main.circuits.none") }}
							</p>
						</div>
						<div v-else class="circuits-tree">
							<div
								v-for="item in flatCircuits"
								:key="item.node.name"
								class="circuit-node-root mb-3"
								:class="[
									'circuit-node--level-' + item.level,
									{ 'circuit-node--selected': selectedCircuitName === item.node.name },
								]"
							>
								<div class="circuit-node-header d-flex justify-content-between align-items-start mb-1">
									<div class="me-2 circuit-node-title">
										<div class="fw-semibold text-truncate">
											{{ item.node.config?.title || item.node.title || item.node.name }}
										</div>
										<div v-if="item.hasLimit" class="small text-muted" >
											<AnimatedNumber :to="item.power" :format="fmtPower" />
											/
											<AnimatedNumber :to="item.maxPower" :format="fmtPower" />
										</div>
									</div>
									<div class="d-flex gap-1 flex-shrink-0">
										<span
											v-if="item.node.dimmed"
											class="badge bg-warning text-dark small"
										>
											{{ $t("main.loadManagement.dimmed") }}
										</span>
										<span
											v-if="item.node.curtailed"
											class="badge bg-secondary small"
										>
											{{ $t("main.loadManagement.curtailed") }}
										</span>
									</div>
								</div>
								<div v-if="item.hasLimit" class="circuit-node-bars mb-1">
									<div class="d-flex justify-content-between small text-muted mb-1">
										<span>{{ $t("main.loadManagement.power") }}</span>
										<span class="text-nowrap">
											0 —
											<AnimatedNumber :to="item.maxPower" :format="fmtPower" />
										</span>
									</div>
									<div
										class="progress circuit-progress"
										role="progressbar"
										:aria-valuenow="item.power"
										aria-valuemin="0"
										:aria-valuemax="item.maxPower"
									>
										<div
											class="progress-bar"
											:class="{
												'circuit-bar-fill': true,
												'circuit-bar-fill--warning': item.power >= item.maxPower,
											}"
											:style="{ width: item.usagePercent + '%' }"
										>
										</div>
									</div>
								</div>
								<ul v-if="item.loadpoints.length" class="list-unstyled mb-1 small circuit-node-loadpoints">
									<li
										v-for="lp in item.loadpoints"
										:key="lp.id"
										class="d-flex justify-content-between align-items-center py-1"
									>
										<span class="text-truncate me-2">{{ lp.displayTitle }}</span>
										<span class="text-nowrap">
											<AnimatedNumber :to="lp.chargePower" :format="fmtPower" />
										</span>
									</li>
								</ul>
							</div>
							<div v-if="ungroupedLoadpoints.length" class="mt-3">
								<h6 class="text-muted small mb-2">
									{{ $t("main.circuits.ungrouped") }}
								</h6>
								<ul class="list-unstyled mb-0 small">
									<li
										v-for="lp in ungroupedLoadpoints"
										:key="lp.id"
										class="d-flex justify-content-between align-items-center py-1"
									>
										<span class="text-truncate me-2">{{ lp.displayTitle }}</span>
										<span class="text-nowrap">
											<AnimatedNumber :to="lp.chargePower" :format="fmtPower" />
										</span>
									</li>
								</ul>
							</div>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="close">
							{{ $t("main.circuits.close") }}
						</button>
					</div>
				</div>
			</div>
		</div>
	</teleport>
</template>

<script lang="ts">
import {
	defineComponent,
	type PropType,
	computed,
	nextTick,
	ref,
	watch,
} from "vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import AnimatedNumber from "../Helper/AnimatedNumber.vue";
import type { Circuit, UiLoadpoint } from "@/types/evcc";
import { buildCircuitsTree } from "@/composables/useCircuitsTree";

export default defineComponent({
	name: "CircuitsModal",
	components: {
		AnimatedNumber,
	},
	mixins: [formatter],
	props: {
		show: {
			type: Boolean,
			default: false,
		},
		circuits: {
			type: Object as PropType<Record<string, Circuit> | undefined>,
			default: undefined,
		},
		loadpoints: {
			type: Array as PropType<UiLoadpoint[]>,
			default: () => [],
		},
		selectedCircuitName: {
			type: String as PropType<string | undefined>,
		},
	},
	emits: ["update:show"],
	setup(
		props: {
			show: boolean;
			circuits?: Record<string, Circuit>;
			loadpoints: UiLoadpoint[];
			selectedCircuitName?: string;
		},
		{ emit }: { emit: (event: "update:show", value: boolean) => void }
	) {
		const tree = computed(() =>
			buildCircuitsTree(props.circuits || {}, props.loadpoints)
		);

		const flatCircuits = computed(() => tree.value.flat);
		const ungroupedLoadpoints = computed(
			() => tree.value.ungroupedLoadpoints
		);
		const hasTree = computed<boolean>(() => flatCircuits.value.length > 0);

		const dialog = ref<HTMLElement | null>(null);

		watch(
			() => props.show,
			(show) => {
				if (show) {
					document.body.style.overflow = "hidden";
					nextTick(() => {
						dialog.value?.focus();
					});
				} else {
					document.body.style.overflow = "";
				}
			}
		);

		const close = () => {
			emit("update:show", false);
		};

		return {
			flatCircuits,
			ungroupedLoadpoints,
			hasTree,
			close,
			dialog,
		};
	},
	methods: {
		fmtPower(value: number): string {
			return this.fmtW(value, POWER_UNIT.KW);
		},
	},
});
</script>

<style scoped>
.circuits-modal-backdrop {
	position: fixed;
	inset: 0;
	background-color: rgba(0, 0, 0, 0.5);
	z-index: 1050;
	display: flex;
	align-items: center;
	justify-content: center;
}

.circuits-modal-dialog {
	max-width: 900px;
	width: 100%;
	margin: 1rem;
	max-height: 100vh;
}

.circuits-modal-dialog .modal-content {
	max-height: 100vh;
	display: flex;
	flex-direction: column;
}

.circuits-modal-dialog .modal-body {
	overflow-y: auto;
}

.circuits-modal-dialog .modal-body,
.circuits-modal-dialog .modal-header,
.circuits-modal-dialog .modal-footer {
	padding-left: calc(var(--bs-modal-padding, 1rem) + 2px);
	padding-right: calc(var(--bs-modal-padding, 1rem) + 2px);
}

/* Mobile-friendly full-screen dialog */
@media (max-width: 575.98px) {
	.circuits-modal-backdrop {
		align-items: stretch;
	}

	.circuits-modal-dialog {
		max-width: 100%;
		margin: 0;
		height: 100%;
	}

	.circuits-modal-dialog .modal-content {
		border-radius: 0;
		height: 100%;
	}

	.circuits-modal-dialog .modal-header,
	.circuits-modal-dialog .modal-footer {
		padding-left: 0.75rem;
		padding-right: 0.75rem;
	}

	.circuits-modal-dialog .modal-body {
		padding-left: 0.75rem;
		padding-right: 0.75rem;
		overflow-y: auto;
	}

	.circuit-node-root {
		padding-left: 0.75rem;
		padding-right: 0.75rem;
	}

	.circuit-node--level-1 {
		margin-left: 0.5rem;
		margin-right: 0.5rem;
	}

	.circuit-node--level-2 {
		margin-left: 1.0rem;
	}

	.circuit-node--level-3 {
		margin-left: 1.5rem;
	}
}

.circuit-node-root {
	border-radius: 1rem;
	padding: 0.75rem 1rem;
	background-color: var(--evcc-box);
	min-width: 0;
}

.circuit-node--level-1 {
	margin-left: 0.75rem;
	margin-right: 0.75rem;
}

.circuit-node--level-2 {
	margin-left: 1.5rem;
}

.circuit-node--level-3 {
	margin-left: 2.25rem;
}

.circuit-node--selected {
	box-shadow: 0 0 0 2px var(--evcc-green);
}

.circuit-node-header {
	min-width: 0;
}

.circuit-node-title {
	min-width: 0;
	flex: 1 1 auto;
}

.circuit-progress {
	height: 10px;
	background: var(--evcc-background);
}

.circuit-node-loadpoints {
	border-top: 1px solid var(--evcc-gray);
	margin-top: 0.5rem;
	padding-top: 0.25rem;
}

.circuit-bar-fill {
	background-color: var(--evcc-dark-green);
}

.circuit-bar-fill--warning {
	background-color: var(--evcc-orange);
}
</style>

