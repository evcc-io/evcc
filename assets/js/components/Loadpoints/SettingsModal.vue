<template>
	<GenericModal
		:id="`loadpointSettingsModal_${id}`"
		ref="modal"
		:title="$t('main.loadpointSettings.title', [loadpoint?.title])"
		size="xl"
		data-testid="loadpoint-settings-modal"
		@open="modalVisible"
		@closed="modalInvisible"
	>
		<div class="container">
			<SmartCostLimit
				:current-limit="loadpoint?.smartCostLimit ?? null"
				:last-limit="loadpoint?.lastSmartCostLimit"
				:smart-cost-type="smartCostType"
				:currency="currency"
				is-loadpoint
				:loadpoint-id="id"
				:multiple-loadpoints="multipleLoadpoints"
				:possible="smartCostAvailable"
				:tariff="forecast?.planner"
				class="mt-2 mb-4"
			/>
			<SmartFeedInPriority
				:current-limit="loadpoint?.smartFeedInPriorityLimit ?? null"
				:last-limit="loadpoint?.lastSmartFeedInPriorityLimit"
				:currency="currency"
				:loadpoint-id="id"
				:multiple-loadpoints="multipleLoadpoints"
				:possible="smartFeedInPriorityAvailable"
				:tariff="forecast?.feedin"
				class="mt-2 mb-4"
			/>
			<LoadpointSettingsBatteryBoost
				v-if="batteryBoostAvailable"
				v-bind="batteryBoostProps"
				class="mt-2"
				@batteryboostlimit-updated="setBatteryBoostLimit"
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
					<p v-if="!loadpoint?.chargerPhases1p3p" class="mt-0 mb-2">
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
							@change="setPhasesConfigured"
						/>
						<label class="form-check-label" :for="formId(`phases_${phases}`)">
							{{ $t(`main.loadpointSettings.phasesConfigured.phases_${phases}`) }}
							<small v-if="phases > 0">
								{{
									$t(
										`main.loadpointSettings.phasesConfigured.phases_${phases}_hint`,
										{
											min: fmtPhasePower(minCurrent, phases),
											max: fmtPhasePower(maxCurrent, phases),
										}
									)
								}}
							</small>
						</label>
					</div>
				</div>
			</div>

			<div v-if="!switchDevice" class="mb-3 row">
				<label :for="formId('maxcurrent')" class="col-sm-4 col-form-label pt-0 pt-sm-2">
					{{ $t("main.loadpointSettings.maxCurrent.label") }}
				</label>
				<div class="col-sm-8 col-lg-4 pe-0 d-flex align-items-center">
					<select
						:id="formId('maxcurrent')"
						v-model.number="selectedMaxCurrent"
						class="form-select form-select-sm"
						@change="setMaxCurrent"
					>
						<option
							v-for="{ value, name } in maxCurrentOptions"
							:key="value"
							:value="value"
						>
							{{ name }}
						</option>
					</select>
				</div>
			</div>

			<div class="mb-3 row">
				<label :for="formId('mincurrent')" class="col-sm-4 col-form-label pt-0 pt-sm-2">
					{{ $t("main.loadpointSettings.minCurrent.label") }}
				</label>
				<div class="col-sm-8 col-lg-4 pe-0 d-flex align-items-center">
					<select
						:id="formId('mincurrent')"
						v-model.number="selectedMinCurrent"
						class="form-select form-select-sm"
						@change="setMinCurrent"
					>
						<option
							v-for="{ value, name } in minCurrentOptions"
							:key="value"
							:value="value"
						>
							{{ name }}
						</option>
					</select>
				</div>
			</div>

			<h6>
				{{ $t("main.loadpointSettings.priority.heading") }}
			</h6>
			<div class="mb-3 row">
				<label
					:for="formId('prioritystrategy')"
					class="col-sm-4 col-form-label pt-0 pt-sm-2"
				>
					{{ $t("main.loadpointSettings.priorityStrategy.label") }}
				</label>
				<div class="col-sm-8 col-lg-4 pe-0 d-flex align-items-center">
					<select
						:id="formId('prioritystrategy')"
						v-model="selectedPriorityStrategy"
						class="form-select form-select-sm"
						@change="setPriorityStrategy"
					>
						<option
							v-for="{ value, name } in priorityStrategyOptions"
							:key="value"
							:value="value"
						>
							{{ name }}
						</option>
					</select>
				</div>
				<div class="col-sm-8 offset-sm-4 pe-0">
					<small class="text-muted">
						{{ $t("main.loadpointSettings.priorityStrategy.help") }}
					</small>
				</div>
			</div>
			<div v-if="priorityHysteresisAvailable" class="mb-3 row">
				<label :for="formId('prioritybasis')" class="col-sm-4 col-form-label pt-0 pt-sm-2">
					{{ $t("main.loadpointSettings.priorityBasis.label") }}
				</label>
				<div class="col-sm-8 col-lg-4 pe-0 d-flex align-items-center">
					<select
						:id="formId('prioritybasis')"
						v-model="selectedPriorityBasis"
						class="form-select form-select-sm"
						@change="setPriorityBasis"
					>
						<option
							v-for="{ value, name } in priorityBasisOptions"
							:key="value"
							:value="value"
						>
							{{ name }}
						</option>
					</select>
				</div>
				<div class="col-sm-8 offset-sm-4 pe-0">
					<small class="text-muted">
						{{ $t("main.loadpointSettings.priorityBasis.help") }}
					</small>
				</div>
			</div>
			<div v-if="priorityHysteresisAvailable" class="mb-3 row">
				<label
					:for="formId('priorityhysteresis')"
					class="col-sm-4 col-form-label pt-0 pt-sm-2"
				>
					{{ $t("main.loadpointSettings.priorityHysteresis.label") }}
				</label>
				<div class="col-sm-8 col-lg-4 pe-0 d-flex align-items-center">
					<input
						:id="formId('priorityhysteresis')"
						v-model.number="selectedPriorityHysteresis"
						class="form-control form-control-sm"
						type="number"
						min="0"
						max="99"
						step="1"
						@change="setPriorityHysteresis"
					/>
					<span class="ms-2">{{ priorityHysteresisUnit }}</span>
				</div>
				<div class="col-sm-8 offset-sm-4 pe-0">
					<small class="text-muted">
						{{ $t("main.loadpointSettings.priorityHysteresis.help") }}
					</small>
				</div>
			</div>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import collector from "@/mixins/collector.ts";
import formatter from "@/mixins/formatter";
import GenericModal from "../Helper/GenericModal.vue";
import SmartCostLimit from "../Tariff/SmartCostLimit.vue";
import SmartFeedInPriority from "../Tariff/SmartFeedInPriority.vue";
import SettingsBatteryBoost from "./SettingsBatteryBoost.vue";
import { defineComponent, type PropType } from "vue";
import {
	PHASES,
	CURRENCY,
	SMART_COST_TYPE,
	PRIORITY_STRATEGY,
	PRIORITY_BASIS,
	type Forecast,
	type UiLoadpoint,
} from "@/types/evcc";
import api from "@/api";

const V = 230;

const range = (start: number, stop: number, step = -1) =>
	Array.from({ length: (stop - start) / step + 1 }, (_, i) => start + i * step);

const insertSorted = (arr: number[], num: number) => {
	const uniqueSet = new Set(arr);
	uniqueSet.add(num);
	return [...uniqueSet].sort((a, b) => b - a);
};

// TODO: add max physical current to loadpoint (config ui) and only allow user to select values in side that range (main ui, here)
const MAX_CURRENT = 64;

const { AUTO, THREE_PHASES, ONE_PHASE } = PHASES;

export default defineComponent({
	name: "LoadpointSettingsModal",
	components: {
		GenericModal,
		SmartCostLimit,
		SmartFeedInPriority,
		LoadpointSettingsBatteryBoost: SettingsBatteryBoost,
	},
	mixins: [formatter, collector],
	props: {
		loadpoints: { type: Array as PropType<UiLoadpoint[]>, default: () => [] },
		batteryConfigured: Boolean,
		smartCostType: String as PropType<SMART_COST_TYPE>,
		smartCostAvailable: Boolean,
		smartFeedInPriorityAvailable: Boolean,
		tariffGrid: Number,
		currency: String as PropType<CURRENCY>,
		multipleLoadpoints: Boolean,
		forecast: Object as PropType<Forecast>,
	},
	data() {
		return {
			id: undefined as string | undefined,
			selectedMaxCurrent: undefined as number | undefined,
			selectedMinCurrent: undefined as number | undefined,
			selectedPhases: undefined as number | undefined,
			selectedPriorityStrategy: PRIORITY_STRATEGY.STATIC as PRIORITY_STRATEGY,
			selectedPriorityBasis: PRIORITY_BASIS.PERCENT as PRIORITY_BASIS,
			selectedPriorityHysteresis: 0 as number,
			isModalVisible: false,
		};
	},
	computed: {
		loadpoint() {
			return this.loadpoints.find((loadpoint) => loadpoint.id === this.id);
		},
		maxCurrent() {
			return this.loadpoint?.maxCurrent;
		},
		minCurrent() {
			return this.loadpoint?.minCurrent;
		},
		switchDevice() {
			return this.loadpoint?.chargerFeatureSwitchDevice;
		},
		batteryBoostLimit() {
			return this.loadpoint?.batteryBoostLimit;
		},
		phasesConfigured() {
			return this.loadpoint?.phasesConfigured;
		},
		phasesOptions() {
			if (this.loadpoint?.chargerSinglePhase) {
				return [];
			}
			if (this.loadpoint?.chargerPhases1p3p) {
				// automatic switching
				return [AUTO, THREE_PHASES, ONE_PHASE];
			}
			// 1p or 3p possible
			return [THREE_PHASES, ONE_PHASE];
		},
		batteryBoostProps() {
			return this.collectProps(SettingsBatteryBoost);
		},
		maxPhases() {
			if (this.loadpoint?.chargerPhases1p3p && this.phasesConfigured === AUTO) {
				return THREE_PHASES;
			}
			return this.phasesConfigured;
		},
		minPhases() {
			if (this.loadpoint?.chargerPhases1p3p && this.phasesConfigured === AUTO) {
				return ONE_PHASE;
			}
			return this.phasesConfigured;
		},
		minCurrentOptions() {
			const opt1 = [...range(Math.floor(this.maxCurrent ?? 0), 1), 0.5, 0.25, 0.125];
			// ensure that current value is always included
			const opt2 = insertSorted(opt1, this.minCurrent ?? 0);
			return opt2.map((value) => this.currentOption(value, value === 6, this.minPhases));
		},
		maxCurrentOptions() {
			const opt1 = range(MAX_CURRENT, Math.ceil(this.minCurrent ?? 0));
			// ensure that current value is always included
			const opt2 = insertSorted(opt1, this.maxCurrent ?? 0);
			return opt2.map((value) => this.currentOption(value, value === 16, this.maxPhases));
		},
		batteryBoostAvailable() {
			return this.batteryConfigured;
		},
		priorityStrategy(): PRIORITY_STRATEGY {
			// published state sends "" for the static (default) strategy
			return this.loadpoint?.priorityStrategy || PRIORITY_STRATEGY.STATIC;
		},
		priorityBasis(): PRIORITY_BASIS {
			// published state sends "" for the percent (default) basis
			return this.loadpoint?.priorityBasis || PRIORITY_BASIS.PERCENT;
		},
		priorityHysteresis(): number {
			return this.loadpoint?.priorityHysteresis ?? 0;
		},
		priorityStrategyOptions(): { value: PRIORITY_STRATEGY; name: string }[] {
			return [
				{
					value: PRIORITY_STRATEGY.STATIC,
					name: this.$t("main.loadpointSettings.priorityStrategy.static"),
				},
				{
					value: PRIORITY_STRATEGY.SOC,
					name: this.$t("main.loadpointSettings.priorityStrategy.soc"),
				},
				{
					value: PRIORITY_STRATEGY.DEFICIT,
					name: this.$t("main.loadpointSettings.priorityStrategy.deficit"),
				},
			];
		},
		priorityBasisOptions(): { value: PRIORITY_BASIS; name: string }[] {
			return [
				{
					value: PRIORITY_BASIS.PERCENT,
					name: this.$t("main.loadpointSettings.priorityBasis.percent"),
				},
				{
					value: PRIORITY_BASIS.ENERGY,
					name: this.$t("main.loadpointSettings.priorityBasis.energy"),
				},
			];
		},
		priorityHysteresisAvailable(): boolean {
			// hysteresis only affects soc/deficit sub-ordering, not static priority
			return this.selectedPriorityStrategy !== PRIORITY_STRATEGY.STATIC;
		},
		priorityHysteresisUnit(): string {
			return this.selectedPriorityBasis === PRIORITY_BASIS.ENERGY ? "kWh" : "%";
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
		priorityStrategy(value) {
			this.selectedPriorityStrategy = value;
		},
		priorityBasis(value) {
			this.selectedPriorityBasis = value;
		},
		priorityHysteresis(value) {
			this.selectedPriorityHysteresis = value;
		},
	},
	methods: {
		open(loadpointId: string) {
			this.id = loadpointId;
			this.selectedPhases = this.phasesConfigured;
			this.selectedMaxCurrent = this.maxCurrent;
			this.selectedMinCurrent = this.minCurrent;
			this.selectedPriorityStrategy = this.priorityStrategy;
			this.selectedPriorityBasis = this.priorityBasis;
			this.selectedPriorityHysteresis = this.priorityHysteresis;
			const modalRef = this.$refs["modal"] as InstanceType<typeof GenericModal> | undefined;
			modalRef?.open();
		},
		apiPath(func: string) {
			return "loadpoints/" + this.id + "/" + func;
		},
		fmtPhasePower(current?: number, phases?: PHASES) {
			return this.fmtW(V * (current || 0) * (phases || 0));
		},
		formId(name: string) {
			return `loadpoint_${this.id}_${name}`;
		},
		setMaxCurrent() {
			api.post(this.apiPath("maxcurrent") + "/" + this.selectedMaxCurrent);
		},
		setMinCurrent() {
			api.post(this.apiPath("mincurrent") + "/" + this.selectedMinCurrent);
		},
		setPhasesConfigured() {
			api.post(this.apiPath("phases") + "/" + this.selectedPhases);
		},
		setBatteryBoostLimit(limit: number) {
			api.post(this.apiPath("batteryboostlimit") + "/" + limit);
		},
		setPriorityStrategy() {
			api.post(this.apiPath("prioritystrategy") + "/" + this.selectedPriorityStrategy);
		},
		setPriorityBasis() {
			api.post(this.apiPath("prioritybasis") + "/" + this.selectedPriorityBasis);
		},
		setPriorityHysteresis() {
			const value = Math.min(
				99,
				Math.max(0, Math.round(this.selectedPriorityHysteresis || 0))
			);
			this.selectedPriorityHysteresis = value;
			api.post(this.apiPath("priorityhysteresis") + "/" + value);
		},
		currentOption(current: number, isDefault: boolean, phases?: number) {
			const kw = this.fmtPhasePower(current, phases);
			let name = `${this.fmtNumber(current, undefined)} A (${kw})`;
			if (isDefault) {
				name += ` [${this.$t("main.loadpointSettings.default")}]`;
			}
			return { value: current, name };
		},
		modalVisible() {
			this.isModalVisible = true;
		},
		modalInvisible() {
			this.isModalVisible = false;
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
