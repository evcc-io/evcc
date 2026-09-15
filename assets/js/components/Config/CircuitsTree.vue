<template>
	<div data-testid="circuit-node">
		<div class="d-flex align-items-stretch row-spacing">
			<template v-if="depth > 0">
				<span v-for="i in depth - 1" :key="i" class="tree-col">
					<span class="tree-line" />
				</span>
				<span class="tree-col">
					<span class="tree-line" />
					<span class="tree-knick" />
				</span>
			</template>
			<DeviceRefBox compact class="flex-grow-1" @edit="editCircuit">
				<span class="d-flex align-items-center gap-2">
					<span class="fw-bold">{{ circuitsTree?.deviceTitle }}</span>
					<span class="ms-auto me-2 evcc-gray small tabular">{{ valueLabel }}</span>
				</span>
			</DeviceRefBox>
		</div>

		<CircuitsTree
			v-for="child in circuitsTree?.children"
			:key="child.name"
			:circuits-tree="child"
			:depth="depth + 1"
			:meters="meters"
			:gridMeter="gridMeter"
		/>

		<div class="d-flex align-items-stretch row-spacing">
			<span v-for="i in depth" :key="i" class="tree-col">
				<span class="tree-line" />
			</span>
			<span class="tree-col">
				<span class="tree-line tree-line--half" />
				<span class="tree-knick" />
			</span>
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
import DeviceRefBox from "./DeviceRefBox.vue";
import AddIcon from "../MaterialIcon/Add.vue";
import formatter from "@/mixins/formatter.ts";
import { openModal } from "@/configModal.ts";
import { meterTitle, type ConfigCircuitNode } from "@/utils/circuits.ts";
import { ICON_SIZE, type ConfigMeter } from "@/types/evcc";

export default {
	name: "CircuitsTree",
	mixins: [formatter],
	components: { DeviceRefBox, AddIcon },
	props: {
		circuitsTree: {
			type: Object as PropType<ConfigCircuitNode>,
		},
		/** Nesting depth from root (0 = root, no indentation/lines). */
		depth: { type: Number, default: 0 },
		meters: {
			type: Array as PropType<ConfigMeter[]>,
			default: () => [],
		},
		gridMeter: { type: Object as PropType<ConfigMeter> },
	},
	methods: {
		addSub() {
			openModal("circuit", { parentId: this.circuitsTree?.name });
		},
		editCircuit() {
			openModal("circuit", { id: this.circuitsTree?.id });
		},
	},
	data() {
		return { ICON_SIZE };
	},
	computed: {
		valueLabel(): string {
			if (!this.circuitsTree) return "";
			const maxpower = Number(this.circuitsTree.config.maxpower);
			const maxcurrent = Number(this.circuitsTree.config.maxcurrent);
			const meterRef =
				"meter" in this.circuitsTree.config
					? String(this.circuitsTree.config["meter"])
					: undefined;

			const meter =
				meterRef && meterRef === this.gridMeter?.name
					? this.$t("config.grid.title")
					: meterTitle(this.meters, meterRef);

			const parts: string[] = [];
			if (maxpower > 0) parts.push(this.fmtW(maxpower, this.POWER_UNIT.AUTO));
			if (maxcurrent > 0) parts.push(`${this.fmtNumber(maxcurrent, 0)} A`);
			if (meter) parts.push(meter);

			return parts.join(" · ");
		},
	},
};
</script>

<style scoped>
.row-spacing {
	margin-bottom: 4px;
}

.tree-col {
	width: 22px;
	position: relative;
	flex: 0 0 auto;
}

.tree-line {
	position: absolute;
	left: 10px;
	top: 0;
	bottom: -4px;
	width: 1px;
	background: var(--evcc-gray-25);
}

.tree-line--half {
	top: 0;
	bottom: auto;
	height: 50%;
}

.tree-knick {
	position: absolute;
	left: 10px;
	top: 50%;
	width: 12px;
	height: 1px;
	background: var(--evcc-gray-25);
}
</style>
