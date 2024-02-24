<template>
	<Teleport to="body">
		<div
			:id="`loadpointSettingsModal_${id}`"
			class="modal fade text-dark modal-xl"
			ref="modal"
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
								v-if="isModalVisible && smartCostAvailable"
								v-bind="smartCostLimitProps"
								class="mt-2"
							/>
							<h4 class="d-flex align-items-center mb-3 mt-5 text-evcc">
								{{ $t("main.loadpointSettings.currents") }}
							</h4>
							<div v-if="phasesOptions.length" class="mb-3 row">
								<label
									:for="formId('phases_0')"
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

<script>
import collector from "../mixins/collector";
import formatter from "../mixins/formatter";
import SmartCostLimit from "./SmartCostLimit.vue";
import smartCostAvailable from "../utils/smartCostAvailable";

const V = 230;

const PHASES_AUTO = 0;
const PHASES_1 = 1;
const PHASES_3 = 3;

const range = (start, stop, step = -1) =>
	Array.from({ length: (stop - start) / step + 1 }, (_, i) => start + i * step);

const insertSorted = (arr, num) => {
	const uniqueSet = new Set(arr);
	uniqueSet.add(num);
	return [...uniqueSet].sort((a, b) => b - a);
};

export default {
	name: "LoadpointSettingsModal",
	mixins: [formatter, collector],
	components: { SmartCostLimit },
	props: {
		id: [String, Number],
		phasesConfigured: Number,
		phasesActive: Number,
		chargerPhases1p3p: Boolean,
		chargerPhysicalPhases: Number,
		minSoc: Number,
		maxCurrent: Number,
		minCurrent: Number,
		title: String,
		smartCostLimit: Number,
		smartCostType: String,
		tariffGrid: Number,
		currency: String,
		multipleLoadpoints: Boolean,
	},
	emits: ["phasesconfigured-updated", "maxcurrent-updated", "mincurrent-updated"],
	data: function () {
		return {
			selectedMaxCurrent: this.maxCurrent,
			selectedMinCurrent: this.minCurrent,
			selectedPhases: this.phasesConfigured,
			isModalVisible: false,
		};
	},
	computed: {
		phasesOptions: function () {
			if (this.chargerPhysicalPhases == 1) {
				// known fixed phase configuration, no settings required
				return [];
			}
			if (this.chargerPhases1p3p) {
				// automatic switching
				return [PHASES_AUTO, PHASES_3, PHASES_1];
			}
			// 1p or 3p possible
			return [PHASES_3, PHASES_1];
		},
		maxPower: function () {
			if (this.chargerPhases1p3p) {
				if (this.phasesConfigured === PHASES_AUTO) {
					return this.maxPowerPhases(3);
				}
				if ([PHASES_3, PHASES_1].includes(this.phasesConfigured)) {
					return this.maxPowerPhases(this.phasesConfigured);
				}
			}
			return this.fmtKw(this.maxCurrent * V * this.phasesActive);
		},
		minPower: function () {
			if (this.chargerPhases1p3p) {
				if (this.phasesConfigured === PHASES_AUTO) {
					return this.minPowerPhases(1);
				}
				if ([PHASES_3, PHASES_1].includes(this.phasesConfigured)) {
					return this.minPowerPhases(this.phasesConfigured);
				}
			}
			return this.fmtKw(this.minCurrent * V * this.phasesActive);
		},
		minCurrentOptions: function () {
			const opt1 = [...range(Math.floor(this.maxCurrent), 1), 0.5, 0.25, 0.125];
			// ensure that current value is always included
			const opt2 = insertSorted(opt1, this.minCurrent);
			return opt2.map((value) => this.currentOption(value, value === 6));
		},
		maxCurrentOptions: function () {
			const opt1 = range(32, Math.ceil(this.minCurrent));
			// ensure that current value is always included
			const opt2 = insertSorted(opt1, this.maxCurrent);
			return opt2.map((value) => this.currentOption(value, value === 16));
		},
		smartCostLimitProps: function () {
			return this.collectProps(SmartCostLimit);
		},
		loadpointId: function () {
			return this.id;
		},
		smartCostAvailable() {
			return smartCostAvailable(this.smartCostType);
		},
	},
	watch: {
		maxCurrent: function (value) {
			this.selectedMaxCurrent = value;
		},
		minCurrent: function (value) {
			this.selectedMinCurrent = value;
		},
		phasesConfigured: function (value) {
			this.selectedPhases = value;
		},
		minSoc: function (value) {
			this.selectedMinSoc = value;
		},
	},
	mounted() {
		this.$refs.modal.addEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal.addEventListener("hidden.bs.modal", this.modalInvisible);
	},
	unmounted() {
		this.$refs.modal?.removeEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal?.removeEventListener("hidden.bs.modal", this.modalInvisible);
	},
	methods: {
		maxPowerPhases: function (phases) {
			return this.fmtKw(this.maxCurrent * V * phases);
		},
		minPowerPhases: function (phases) {
			return this.fmtKw(this.minCurrent * V * phases);
		},
		formId: function (name) {
			return `loadpoint_${this.id}_${name}`;
		},
		changeMaxCurrent: function () {
			this.$emit("maxcurrent-updated", this.selectedMaxCurrent);
		},
		changeMinCurrent: function () {
			this.$emit("mincurrent-updated", this.selectedMinCurrent);
		},
		changePhasesConfigured: function () {
			this.$emit("phasesconfigured-updated", this.selectedPhases);
		},
		currentOption: function (value, isDefault) {
			let name = `${this.fmtNumber(value)} A`;
			if (isDefault) {
				name += ` (${this.$t("main.loadpointSettings.default")})`;
			}
			return { value, name };
		},
		modalVisible: function () {
			this.isModalVisible = true;
		},
		modalInvisible: function () {
			this.isModalVisible = false;
		},
	},
};
</script>
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
}

.container h4:first-child {
	margin-top: 0 !important;
}
</style>
