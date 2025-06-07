<template>
	<Teleport to="body">
		<div
			:id="`loadpointSettingsModal_${id}`"
			ref="modal"
			class="modal fade text-dark modal-xl"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
			data-testid="loadpoint-settings-modal"
		>
			<div class="modal-dialog modal-dialog-centered" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							{{ $t("main.loadpointSettings.title", [title]) }}
						</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
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
											{{
												$t(
													"main.loadpointSettings.phasesConfigured.no1p3pSupport"
												)
											}}</small
										>
									</p>
									<div
										v-for="phases in phasesOptions"
										:key="phases"
										class="form-check"
									>
										<input
											:id="formId(`phases_${phases}`)"
											v-model.number="selectedPhases"
											class="form-check-input"
											type="radio"
											:name="formId('phases')"
											:value="phases"
											@change="changePhasesConfigured"
										/>
										<label
											class="form-check-label"
											:for="formId(`phases_${phases}`)"
										>
											{{
												$t(
													`main.loadpointSettings.phasesConfigured.phases_${phases}`
												)
											}}
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
								<label
									:for="formId('maxcurrent')"
									class="col-sm-4 col-form-label pt-0 pt-sm-2"
								>
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
								<label
									:for="formId('mincurrent')"
									class="col-sm-4 col-form-label pt-0 pt-sm-2"
								>
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
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script lang="ts">
import collector from "@/mixins/collector";
import formatter from "@/mixins/formatter";
import SmartCostLimit from "../Tariff/SmartCostLimit.vue";
import smartCostAvailable from "@/utils/smartCostAvailable";
import SettingsBatteryBoost from "./SettingsBatteryBoost.vue";
import { defineComponent, type PropType } from "vue";
import { CURRENCY, PHASES } from "@/types/evcc";

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
	components: { SmartCostLimit, LoadpointSettingsBatteryBoost: SettingsBatteryBoost },
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
		title: String,
		smartCostLimit: { type: Number as PropType<number | null>, default: null },
		smartCostType: String,
		tariffGrid: Number,
		currency: String as PropType<CURRENCY>,
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
		minCurrentOptions() {
			const opt1 = [...range(Math.floor(this.maxCurrent), 1), 0.5, 0.25, 0.125];
			// ensure that current value is always included
			const opt2 = insertSorted(opt1, this.minCurrent);
			return opt2.map((value) => this.currentOption(value, value === 6));
		},
		maxCurrentOptions() {
			const opt1 = range(MAX_CURRENT, Math.ceil(this.minCurrent));
			// ensure that current value is always included
			const opt2 = insertSorted(opt1, this.maxCurrent);
			return opt2.map((value) => this.currentOption(value, value === 16));
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
	mounted() {
		this.$refs["modal"]?.addEventListener("show.bs.modal", this.modalVisible);
		this.$refs["modal"]?.addEventListener("hidden.bs.modal", this.modalInvisible);
	},
	unmounted() {
		this.$refs["modal"]?.removeEventListener("show.bs.modal", this.modalVisible);
		this.$refs["modal"]?.removeEventListener("hidden.bs.modal", this.modalInvisible);
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
		currentOption(value: number, isDefault: boolean) {
			let name = `${this.fmtNumber(value, 0)} A`;
			if (isDefault) {
				name += ` (${this.$t("main.loadpointSettings.default")})`;
			}
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
