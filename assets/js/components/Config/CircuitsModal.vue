<template>
	<GenericModal
		id="circuitsModal"
		config-modal-name="circuits"
		:title="$t('config.circuits.title')"
		data-testid="circuits-modal"
		:autofocus="false"
		@close="cancelSelection"
	>
		<template #default>
			<p>
				{{ $t("config.circuits.description") }}
				<a :href="docsLink" target="_blank">{{ $t("config.general.docsLink") }}</a>
			</p>
			<div :aria-busy="pending" @keydown.esc.stop="cancelSelection">
				<NewDeviceButton
					v-if="circuits.length === 0"
					:title="$t('config.circuits.addMainCircuit')"
					@click="openCircuit()"
				/>
				<div v-else>
					<CircuitsTree
						class="mb-3"
						:circuitsTree="configCircuitTree(circuits)"
						:grid-meter="gridMeter"
						:loadpoints="loadpoints"
						:selected-id="selectedId"
						:dragged-id="draggedId"
						:hover-target="hoverTarget"
						:pending="pending"
						:flash-id="visibleFlashId"
						@select="selectLoadpoint"
						@assign="requestAssignment($event.id, $event.circuit)"
						@transfer="transfer"
						@hover="hoverTarget = $event"
						@dragstart="startDrag"
						@dragend="endDrag"
					/>
				</div>
				<DragDropList
					class="assignment-pool d-flex flex-column gap-2"
					data-testid="assignment-pool"
					:class="{
						'assignment-target': canRemove,
						'assignment-hover': canRemove && hoverTarget === 'unassigned',
					}"
					:values="poolValues"
					transfer
					group="circuit-assignment"
					name="unassigned"
					:disabled="pending"
					:draggable-value="isEditable"
					:aria-label="$t('config.circuits.notAssigned')"
					tabindex="0"
					@click="activatePool"
					@keydown.enter.self.prevent="activatePool"
					@keydown.space.self.prevent="activatePool"
					@transfer="transfer"
					@hover="hoverTarget = $event"
					@dragstart="startDrag"
					@dragend="endDrag"
				>
					<h6 class="small evcc-gray text-uppercase m-0 px-1">
						{{ $t("config.circuits.notAssigned") }}
					</h6>
					<DragDropItem
						v-for="lp in unassignedLoadpoints"
						:key="lp.id ?? lp.name"
						:title="lp.title"
						:data-loadpoint="lp.id"
						interactive
						class="pool-row mb-0 py-0"
						:selected="lp.id === selectedId"
						:disabled="pending || !lp.editable"
						:class="{ 'assignment-dragged': lp.id === draggedId }"
						@activate="selectLoadpoint(lp.id)"
					>
						<span class="d-flex align-items-center gap-2 min-w-0"
							><Heatpump
								v-if="lp.heating"
								class="device-icon"
							/><shopicon-regular-cablecharge v-else class="device-icon" /><span
								class="text-truncate"
								>{{ lp.title }}</span
							></span
						>
					</DragDropItem>
					<div class="d-flex gap-2 align-items-center px-1">
						<span class="small evcc-gray flex-grow-1" aria-live="polite">{{
							poolHint
						}}</span
						><button
							v-if="selectedId !== undefined"
							type="button"
							class="btn btn-sm btn-link"
							@click.stop="cancelSelection"
						>
							{{ $t("config.general.cancel") }}
						</button>
					</div>
				</DragDropList>
				<p v-if="error" role="alert" class="text-danger mt-2">{{ error }}</p>
			</div>
			<div class="d-flex justify-content-end mt-4">
				<button type="button" class="btn btn-outline-primary px-4" data-bs-dismiss="modal">
					{{ $t("config.general.close") }}
				</button>
			</div>
		</template>
	</GenericModal>
</template>

<script lang="ts">
import GenericModal from "../Helper/GenericModal.vue";
import DragDropList from "../Helper/DragDropList.vue";
import DragDropItem from "../Helper/DragDropItem.vue";
import Heatpump from "../VehicleIcon/Heatpump.vue";
import "@h2d2/shopicons/es/regular/cablecharge";
import { docsPrefix } from "@/i18n";
import type { ConfigCircuit, ConfigMeter } from "@/types/evcc";
import CircuitsTree from "./CircuitsTree.vue";
import NewDeviceButton from "./NewDeviceButton.vue";
import { openModal } from "@/configModal.ts";
import { configCircuitTree, type CircuitAssignmentLoadpoint } from "@/utils/circuits.ts";
import type { PropType } from "vue";

export default {
	name: "CircuitsModal",
	components: {
		GenericModal,
		CircuitsTree,
		NewDeviceButton,
		DragDropList,
		DragDropItem,
		Heatpump,
	},
	emits: ["changed", "assign"],
	props: {
		circuits: { type: Array as PropType<ConfigCircuit[]>, required: true },
		gridMeter: { type: Object as PropType<ConfigMeter> },
		loadpoints: { type: Array as PropType<CircuitAssignmentLoadpoint[]>, default: () => [] },
		pending: Boolean,
		error: String,
		revision: Number,
		flashId: Number,
	},
	data() {
		return {
			selectedId: undefined as number | undefined,
			draggedId: undefined as number | undefined,
			hoverTarget: undefined as string | undefined,
			visibleFlashId: undefined as number | undefined,
			flashTimer: undefined as ReturnType<typeof setTimeout> | undefined,
		};
	},
	computed: {
		docsLink(): string {
			return `${docsPrefix()}/features/loadmanagement`;
		},
		unassignedLoadpoints(): CircuitAssignmentLoadpoint[] {
			return this.loadpoints.filter(
				(lp) => !this.circuits.some((c) => c.name === lp.circuit)
			);
		},
		poolValues(): (number | string | undefined)[] {
			return this.unassignedLoadpoints.map((lp) => lp.id ?? lp.name);
		},
		activeLoadpoint(): CircuitAssignmentLoadpoint | undefined {
			return this.loadpoints.find((lp) => lp.id === (this.draggedId ?? this.selectedId));
		},
		canRemove(): boolean {
			return !!this.activeLoadpoint?.circuit;
		},
		poolHint(): string {
			if (this.draggedId !== undefined && this.canRemove)
				return this.$t("config.circuits.dropRemove");
			if (this.selectedId !== undefined)
				return this.$t(
					this.canRemove ? "config.circuits.selectMove" : "config.circuits.selectAssign",
					{ title: this.activeLoadpoint?.title }
				);
			return this.$t("config.circuits.dragHint");
		},
	},
	watch: {
		revision() {
			this.cancelSelection();
			this.visibleFlashId = this.flashId;
			this.flashTimer = setTimeout(() => {
				this.visibleFlashId = undefined;
			}, 1400);
		},
	},
	beforeUnmount() {
		clearTimeout(this.flashTimer);
	},
	methods: {
		isEditable(value: unknown): boolean {
			return this.unassignedLoadpoints.some(
				(lp) => (lp.id ?? lp.name) === value && lp.editable
			);
		},
		selectLoadpoint(id?: number) {
			if (this.pending || !this.loadpoints.some((lp) => lp.id === id && lp.editable)) return;
			this.selectedId = this.selectedId === id ? undefined : id;
		},
		requestAssignment(id: number, circuit: string | null) {
			const lp = this.loadpoints.find((lp) => lp.id === id);
			if (this.pending || !lp?.editable || lp.circuit === circuit) return;
			this.$emit("assign", { id, circuit });
		},
		transfer({ value, to }: { value: number; to: string }) {
			this.requestAssignment(value, to === "unassigned" ? null : to);
		},
		activatePool() {
			if (this.selectedId !== undefined && this.canRemove)
				this.requestAssignment(this.selectedId, null);
		},
		startDrag(id: number) {
			this.selectedId = undefined;
			this.draggedId = id;
		},
		endDrag() {
			this.draggedId = undefined;
			this.hoverTarget = undefined;
		},
		cancelSelection() {
			this.selectedId = undefined;
			this.endDrag();
			this.visibleFlashId = undefined;
			clearTimeout(this.flashTimer);
		},
		configCircuitTree,
		async openCircuit() {
			await openModal("circuit");
		},
	},
};
</script>

<style>
.assignment-target {
	box-shadow: 0 0 0 1px var(--bs-primary-dim);
}
.assignment-hover {
	box-shadow: 0 0 0 2px var(--bs-primary);
	background: color-mix(in srgb, var(--bs-primary) 12%, var(--evcc-box));
}
.assignment-flash {
	animation: assignment-flash 1.4s ease-out;
}
.assignment-dragged {
	opacity: 0.35;
}
@keyframes assignment-flash {
	from {
		background: color-mix(in srgb, var(--bs-primary) 18%, var(--evcc-box));
	}
}
</style>
<style scoped>
.assignment-pool {
	border: 2px dashed var(--evcc-gray-25);
	border-radius: 12px;
	padding: 12px;
}
.pool-row {
	min-height: 44px;
}
.device-icon {
	width: 20px;
	height: 20px;
	flex-shrink: 0;
}
</style>
