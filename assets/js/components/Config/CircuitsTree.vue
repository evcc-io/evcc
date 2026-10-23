<template>
	<div data-testid="circuit-node" class="circuit-branch position-relative">
		<DragDropList
			:values="assignedValues"
			transfer
			group="circuit-assignment"
			:name="circuitsTree?.name"
			:disabled="pending"
			:draggable-value="isEditable"
			@transfer="$emit('transfer', $event)"
			@hover="$emit('hover', $event)"
			@dragstart="$emit('dragstart', $event)"
			@dragend="$emit('dragend')"
		>
			<div class="d-flex align-items-stretch row-spacing">
				<div
					:data-circuit="circuitsTree?.name"
					class="assignment-circuit border rounded d-flex flex-grow-1 min-w-0 align-items-center px-2"
					:class="{ 'assignment-target': isCandidate, 'assignment-hover': isHoverTarget }"
				>
					<button
						type="button"
						class="btn border-0 p-0 d-flex align-items-center gap-2 min-w-0 flex-grow-1 text-start"
						:aria-label="
							$t('config.circuits.destination', { title: circuitsTree?.deviceTitle })
						"
						:disabled="pending"
						@click="activateCircuit"
					>
						<span class="fw-bold text-truncate">{{ circuitsTree?.deviceTitle }}</span>
						<span
							v-if="targetLabel"
							class="ms-auto me-2 text-primary fw-bold small text-truncate"
						>
							{{ targetLabel }}
						</span>
						<span
							v-else
							class="ms-auto me-2 evcc-gray small tabular d-flex gap-1 min-w-0"
						>
							<span class="text-nowrap">{{ limitsLabel }}</span>
							<span v-if="limitsLabel && meterLabel" aria-hidden="true">·</span>
							<span class="text-truncate">{{ meterLabel }}</span>
						</span>
					</button>
					<DeviceCardEditIcon :editable="true" @edit="editCircuit" />
				</div>
			</div>
			<DragDropItem
				v-for="lp in assignedLoadpoints"
				:key="lp.id ?? lp.name"
				:title="lp.title"
				:data-loadpoint="lp.id"
				interactive
				assigned
				:disabled="pending || !lp.editable"
				:selected="selectedId !== undefined && lp.id === selectedId"
				:class="{
					'assignment-flash': flashId !== undefined && lp.id === flashId,
					'assignment-dragged': draggedId !== undefined && lp.id === draggedId,
				}"
				class="assigned-row position-relative py-0"
				@activate="$emit('select', lp.id)"
			>
				<span class="d-flex align-items-center gap-2 min-w-0">
					<Heatpump v-if="lp.heating" class="device-icon" />
					<shopicon-regular-cablecharge v-else class="device-icon" />
					<span class="text-truncate">{{ lp.title }} </span>
				</span>
			</DragDropItem>
		</DragDropList>

		<CircuitsTree
			v-for="child in circuitsTree?.children"
			:key="child.name"
			class="child-circuit"
			:circuits-tree="child"
			:gridMeter="gridMeter"
			:loadpoints="loadpoints"
			:selected-id="selectedId"
			:dragged-id="draggedId"
			:hover-target="hoverTarget"
			:pending="pending"
			:flash-id="flashId"
			:active-loadpoint="activeLoadpoint"
			@select="$emit('select', $event)"
			@assign="$emit('assign', $event)"
			@transfer="$emit('transfer', $event)"
			@hover="$emit('hover', $event)"
			@dragstart="$emit('dragstart', $event)"
			@dragend="$emit('dragend')"
		/>

		<div class="d-flex align-items-stretch row-spacing branch-action position-relative">
			<button
				type="button"
				class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
				data-testid="circuit-add-sub"
				tabindex="0"
				@click="addSub"
			>
				<AddIcon :size="ICON_SIZE.XS" class="flex-shrink-0" />

				{{ $t("config.circuits.addSubCircuit") }}
			</button>
		</div>
	</div>
</template>

<script lang="ts">
import type { PropType } from "vue";
import DeviceCardEditIcon from "./DeviceCardEditIcon.vue";
import DragDropList from "../Helper/DragDropList.vue";
import DragDropItem from "../Helper/DragDropItem.vue";
import Heatpump from "../VehicleIcon/Heatpump.vue";
import "@h2d2/shopicons/es/regular/cablecharge";
import AddIcon from "../MaterialIcon/Add.vue";
import formatter from "@/mixins/formatter.ts";
import { openModal } from "@/configModal.ts";
import type { ConfigCircuitNode, CircuitAssignmentLoadpoint } from "@/utils/circuits.ts";
import { ICON_SIZE, type ConfigMeter } from "@/types/evcc";

export default {
	name: "CircuitsTree",
	mixins: [formatter],
	components: { DeviceCardEditIcon, AddIcon, DragDropList, DragDropItem, Heatpump },
	emits: ["select", "assign", "transfer", "hover", "dragstart", "dragend"],
	props: {
		circuitsTree: {
			type: Object as PropType<ConfigCircuitNode>,
		},

		gridMeter: { type: Object as PropType<ConfigMeter> },
		loadpoints: { type: Array as PropType<CircuitAssignmentLoadpoint[]>, default: () => [] },
		selectedId: Number,
		draggedId: Number,
		hoverTarget: String,
		pending: Boolean,
		flashId: Number,
		activeLoadpoint: Object as PropType<CircuitAssignmentLoadpoint>,
	},
	methods: {
		isEditable(value: unknown): boolean {
			return this.loadpoints.some((lp) => lp.editable && lp.id === value);
		},
		activateCircuit() {
			if (this.pending) return;
			if (this.selectedId !== undefined) {
				if (this.isCandidate)
					this.$emit("assign", { id: this.selectedId, circuit: this.circuitsTree?.name });
				return;
			}
			this.editCircuit();
		},
		addSub() {
			openModal("circuit", { parent: this.circuitsTree?.id });
		},
		editCircuit() {
			openModal("circuit", { id: this.circuitsTree?.id });
		},
	},
	data() {
		return { ICON_SIZE };
	},
	computed: {
		assignedLoadpoints(): CircuitAssignmentLoadpoint[] {
			return this.loadpoints.filter((lp) => lp.circuit === this.circuitsTree?.name);
		},
		assignedValues(): (number | string | undefined)[] {
			return this.assignedLoadpoints.map((lp) => lp.id ?? lp.name);
		},
		isCandidate(): boolean {
			return (
				!!this.activeLoadpoint && this.activeLoadpoint.circuit !== this.circuitsTree?.name
			);
		},
		isHoverTarget(): boolean {
			return this.isCandidate && this.hoverTarget === this.circuitsTree?.name;
		},
		targetLabel(): string {
			if (!this.isCandidate) return "";
			if (this.selectedId !== undefined)
				return this.$t(
					this.activeLoadpoint?.circuit
						? "config.circuits.moveHere"
						: "config.circuits.assignHere"
				);
			return this.isHoverTarget ? this.$t("config.circuits.dropAssign") : "";
		},
		limitsLabel(): string {
			if (!this.circuitsTree) return "";
			const maxpower = Number(this.circuitsTree.config.maxpower);
			const maxcurrent = Number(this.circuitsTree.config.maxcurrent);
			const parts: string[] = [];
			if (maxpower > 0) parts.push(this.fmtW(maxpower, this.POWER_UNIT.AUTO));
			if (maxcurrent > 0) parts.push(`${this.fmtNumber(maxcurrent, 0)} A`);
			return parts.join(" · ");
		},
		meterLabel(): string {
			if (!this.circuitsTree) return "";
			const meterRef = this.circuitsTree.config.meter;
			if (!meterRef) return "";
			return this.$t(
				meterRef === this.gridMeter?.name
					? "config.circuits.meterGrid"
					: "config.circuits.meterDedicated"
			);
		},
	},
};
</script>

<style scoped>
.row-spacing {
	margin-bottom: 8px;
}

.assignment-circuit {
	min-height: 42px;
}
.assigned-row {
	margin-left: 22px;
	min-height: 40px;
}
.circuit-branch::before {
	content: "";
	position: absolute;
	left: 10px;
	top: 42px;
	bottom: 16px;
	border-left: 1px solid var(--evcc-gray-25);
	pointer-events: none;
}
.child-circuit,
.branch-action {
	margin-left: 22px;
}
.assigned-row::after,
.child-circuit::after,
.branch-action::after {
	content: "";
	position: absolute;
	left: -12px;
	width: 12px;
	border-top: 1px solid var(--evcc-gray-25);
	pointer-events: none;
}
.assigned-row::after,
.branch-action::after {
	top: 50%;
}
.child-circuit::after {
	top: 21px;
}
.branch-action {
	min-height: 32px;
	margin-bottom: 0;
}
</style>
