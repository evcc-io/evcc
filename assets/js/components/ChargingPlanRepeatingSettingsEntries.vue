<template>
	<div v-for="(entry, index) in entries" :key="index">
		<ChargingPlanRepeatingSettingsEntry
			class="mb-4"
			:id="index"
			v-bind="entry"
			:socBasedPlanning="socBasedPlanning"
			:rangePerSoc="rangePerSoc"
			@repeating-plan-entry-updated="updateRepeatingPlanEntry"
			@repeating-plan-entry-removed="removeRepeatingPlanEntry"
		/>
	</div>
	<div class="d-flex align-items-baseline">
		<button
			type="button"
			class="d-flex btn btn-sm btn-outline-primary border-0 ps-0 align-items-baseline"
			data-testid="repeating-plan-add"
			@click="addRepeatingPlan"
		>
			<shopicon-regular-plus size="s" class="flex-shrink-0 me-2"></shopicon-regular-plus>
			<p class="mb-0">{{ $t("main.chargingPlan.addRepeatingPlan") }}</p>
		</button>
		<button
			v-if="dataHasChanged"
			type="button"
			class="btn btn-sm btn-outline-primary ms-3 border-0 text-decoration-underline"
			data-testid="plan-apply"
			@click="updateRepeatingPlan"
		>
			{{ $t("main.chargingPlan.update") }}
		</button>
	</div>
</template>

<script>
import api from "../api";
import ChargingPlanRepeatingSettingsEntry from "./ChargingPlanRepeatingSettingsEntry.vue";

const DEFAULT_WEEKDAYS = [0];
const DEFAULT_TARGET_TIME = "12:00";
const DEFAULT_TARGET_SOC = 80;

export default {
	name: "ChargingPlanRepeatingSettingsEntries",
	components: {
		ChargingPlanRepeatingSettingsEntry,
	},
	props: {
		id: Number,
		socBasedPlanning: Boolean,
		rangePerSoc: Number,
	},
	emits: ["repeating-plan-updated"],
	data: function () {
		return {
			entries: [],
			initialEntries: [],
		};
	},
	computed: {
		dataHasChanged: function () {
			return JSON.stringify(this.initialEntries) !== JSON.stringify(this.entries);
		},
	},
	methods: {
		addRepeatingPlan: function () {
			this.entries.push({
				weekdays: DEFAULT_WEEKDAYS,
				time: DEFAULT_TARGET_TIME,
				soc: DEFAULT_TARGET_SOC,
				active: false,
			});
		},
		updateRepeatingPlan: function () {
			this.initialEntries = [...this.entries]; // clone array
			this.$emit("repeating-plan-updated", this.initialEntries);
		},
		updateRepeatingPlanEntry: function (newData) {
			this.entries[newData.id] = {
				weekdays: newData.weekdays,
				time: newData.time,
				soc: newData.soc,
				active: newData.active,
			};
		},
		removeRepeatingPlanEntry: function (index) {
			this.entries.splice(index, 1);
		},
	},
};
</script>
