<template>
	<GenericModal
		:id="`loadpointSettingsModal_${id}`"
		:title="$t('main.loadpointSettings.title', [title])"
		size="xl"
		data-testid="loadpoint-settings-modal"
		@open="modalVisible"
		@closed="modalInvisible"
	>
		<div class="container">
			<SmartCostLimit
				v-if="isModalVisible"
				v-bind="smartCostLimitProps"
				:possible="smartCostAvailable"
				:smartCostLimit="smartCostLimit"
				class="mt-2"
			/>
			<LoadpointSettingsBatteryBoost
				v-if="batteryBoostAvailable"
				v-bind="batteryBoostProps"
				class="mt-2"
				@batteryboost-updated="changeBatteryBoost"
			/>
			<h6>
				{{ $t("main.loadpointSettings.currents") }}
			</h6>
			<div v-if="phasesOptions.length" class="mb-3 row">
				<label
					:for="formId(`phases_${phasesOptions[0]}`)"
					class="col-sm-4 col-form-label pt-0"
				>
					{{ $t("main.loadpointSettings.phasesConfigured.label") }}
				</label>
				<div class="col-sm-8 pe-0">
					<p v-if="!chargerPhases1p3p" class="mt-0 mb-2">
						<small>
							{{ $t("main.loadpointSettings.phasesConfigured.no1p3pSupport") }}</small
						>
					</p>
					<div v-for="phases in phasesOptions" :key="phases" class="form-check">
						<input
							:id="formId(`phases_${phases}`)"
							v-model.number="selectedPhases"
							class="form-check-input"
							type="radio"
							:name="formId('phases')"
							:value="phases"
							@change="changePhasesConfigured"
						/>
						<label class="form-check-label" :for="formId(`phases_${phases}`)">
							{{ $t(`main.loadpointSettings.phasesConfigured.phases_${phases}`) }}
							<small v-if="phases > 0">
								{{
									$t(
										`main.loadpointSettings.phasesConfigured.phases_${phases}_hint`,
										{
											min: minPowerPhases(phases),
											max: maxPowerPhases(phases),
										}
									)
								}}
							</small>
						</label>
					</div>
				</div>
			</div>

			<div class="mb-3 row">
				<label :for="formId('maxcurrent')" class="col-sm-4 col-form-label pt-0 pt-sm-2">
					{{ $t("main.loadpointSettings.maxCurrent.label") }}
				</label>
				<div class="col-sm-8 pe-0 d-flex align-items-center">
					<select
						:id="formId('maxcurrent')"
						v-model.number="selectedMaxCurrent"
						class="form-select form-select-sm w-50"
						@change="changeMaxCurrent"
					>
						<option
							v-for="{ value, name } in maxCurrentOptions"
							:key="value"
							:value="value"
						>
							{{ name }}
						</option>
					</select>
					<small class="ms-3">~ {{ maxPower }}</small>
				</div>
			</div>

			<div class="mb-3 row">
				<label :for="formId('mincurrent')" class="col-sm-4 col-form-label pt-0 pt-sm-2">
					{{ $t("main.loadpointSettings.minCurrent.label") }}
				</label>
				<div class="col-sm-8 pe-0 d-flex align-items-center">
					<select
						:id="formId('mincurrent')"
						v-model.number="selectedMinCurrent"
						class="form-select form-select-sm w-50"
						@change="changeMinCurrent"
					>
						<option
							v-for="{ value, name } in minCurrentOptions"
							:key="value"
							:value="value"
						>
							{{ name }}
						</option>
					</select>
					<small class="ms-3">~ {{ minPower }}</small>
				</div>
			</div>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import collector from "@/mixins/collector";
import formatter from "@/mixins/formatter";
import SmartCostLimit from "../Tariff/SmartCostLimit.vue";
import smartCostAvailable from "@/utils/smartCostAvailable";
import SettingsBatteryBoost from "./SettingsBatteryBoost.vue";
import GenericModal from "../Helper/GenericModal.vue";
import { defineComponent, type PropType } from "vue";
import { PHASES } from "@/types/evcc";

const V = 230;

const range = (start: number, end: number) => {
	return Array.from({ length: end - start + 1 }, (_, i) => start + i);
};

const insertSorted = (arr: number[], num: number) => {
	const uniqueSet = new Set(arr);
	uniqueSet.add(num);
	return [...uniqueSet].sort((a, b) => b - a);
};

const { AUTO, THREE_PHASES, ONE_PHASE } = PHASES;

export default defineComponent({
	name: "LoadpointSettingsModal",
	components: {
		SmartCostLimit,
		LoadpointSettingsBatteryBoost: SettingsBatteryBoost,
		GenericModal,
	},
	mixins: [formatter, collector],
	props: {
		id: [String, Number],
		phasesConfigured: { type: Number, default: 0 },
		chargerPhases1p3p: Boolean,
		chargerSinglePhase: Boolean,
		batteryBoost: Boolean,
		batteryBoostAvailable: Boolean,
		mode: String,
		minSoc: Number,
		maxCurrent: { type: Number, default: 0 },
		minCurrent: { type: Number, default: 0 },
		minPhysicalCurrent: Number,
		maxPhysicalCurrent: Number,
		title: String,
		smartCostLimit: { type: Number as PropType<number | null>, default: null },
		smartCostType: String,
		tariffGrid: Number,
		currency: String,
		multipleLoadpoints: Boolean,
	},
	emits: [
		"phasesconfigured-updated",
		"maxcurrent-updated",
		"mincurrent-updated",
		"batteryboost-updated",
	],
	data() {
		return {
			selectedMaxCurrent: this.maxCurrent,
			selectedMinCurrent: this.minCurrent,
			selectedPhases: this.phasesConfigured,
			isModalVisible: false,
		};
	},
	computed: {
		phasesOptions() {
			if (this.chargerSinglePhase) {
				return [];
			}
			if (this.chargerPhases1p3p) {
				// automatic switching
				return [AUTO, THREE_PHASES, ONE_PHASE];
			}
			// 1p or 3p possible
			return [THREE_PHASES, ONE_PHASE];
		},
		batteryBoostProps() {
			return this.collectProps(SettingsBatteryBoost);
		},
		maxPower() {
			if (this.chargerPhases1p3p) {
				if (this.phasesConfigured === AUTO) {
					return this.maxPowerPhases(THREE_PHASES);
				}
				if ([THREE_PHASES, ONE_PHASE].includes(this.phasesConfigured)) {
					return this.maxPowerPhases(this.phasesConfigured);
				}
			}
			return this.fmtW(this.maxCurrent * V * this.phasesConfigured);
		},
		minPower() {
			if (this.chargerPhases1p3p) {
				if (this.phasesConfigured === AUTO) {
					return this.minPowerPhases(ONE_PHASE);
				}
				if ([THREE_PHASES, ONE_PHASE].includes(this.phasesConfigured)) {
					return this.minPowerPhases(this.phasesConfigured);
				}
			}
			return this.fmtW(this.minCurrent * V * this.phasesConfigured);
		},
		currentOptions() {
			const low = [0.125, 0.25, 0.5, 1, 2, 3, 4, 5]; // mostly heating
			const regular = range(6, 64); // ac charging
			const high = [75, 100, 150, 200, 250, 300, 400, 500, 600]; // dc charging
			return [...low, ...regular, ...high].filter(this.isInRange);
		},
		minCurrentOptions() {
			const opt1 = this.currentOptions.filter((value) => value <= this.maxCurrent);
			return insertSorted(opt1, this.minCurrent).map(this.currentOption);
		},
		maxCurrentOptions() {
			const opt1 = this.currentOptions.filter((value) => value >= this.minCurrent);
			return insertSorted(opt1, this.maxCurrent).map(this.currentOption);
		},
		smartCostLimitProps() {
			return this.collectProps(SmartCostLimit);
		},
		loadpointId() {
			return this.id;
		},
		smartCostAvailable() {
			return smartCostAvailable(this.smartCostType);
		},
	},
	watch: {
		maxCurrent(value) {
			this.selectedMaxCurrent = value;
		},
		minCurrent(value) {
			this.selectedMinCurrent = value;
		},
		phasesConfigured(value) {
			this.selectedPhases = value;
		},
	},
	methods: {
		maxPowerPhases(phases: PHASES) {
			return this.fmtW(this.maxCurrent * V * phases);
		},
		minPowerPhases(phases: PHASES) {
			return this.fmtW(this.minCurrent * V * phases);
		},
		formId(name: string) {
			return `loadpoint_${this.id}_${name}`;
		},
		changeMaxCurrent() {
			this.$emit("maxcurrent-updated", this.selectedMaxCurrent);
		},
		changeMinCurrent() {
			this.$emit("mincurrent-updated", this.selectedMinCurrent);
		},
		changePhasesConfigured() {
			this.$emit("phasesconfigured-updated", this.selectedPhases);
		},
		currentOption(value: number) {
			const digits = value < 1 ? undefined : 0;
			const name = `${this.fmtNumber(value, digits)} A`;
			return { value, name };
		},
		modalVisible() {
			this.isModalVisible = true;
		},
		modalInvisible() {
			this.isModalVisible = false;
		},
		changeBatteryBoost(boost: boolean) {
			this.$emit("batteryboost-updated", boost);
		},
		isInRange(value: number) {
			const min = this.minPhysicalCurrent ?? 0;
			const max = this.maxPhysicalCurrent ?? 64;
			return min <= value && value <= max;
		},
	},
});
</script>
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
}

.container h4:first-child {
	margin-top: 0 !important;
}

.custom-select-inline {
	display: inline-block !important;
}
</style>
