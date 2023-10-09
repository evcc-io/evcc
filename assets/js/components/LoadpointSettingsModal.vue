<template>
	<Teleport to="body">
		<div
			:id="`loadpointSettingsModal_${id}`"
			class="modal fade text-dark"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
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
							<h4
								v-if="showConfigurablePhases || showCurrentSettings"
								class="d-flex align-items-center mb-3 mt-4 text-evcc"
							>
								{{ $t("main.loadpointSettings.currents") }}
								<shopicon-bold-lightning class="ms-1"></shopicon-bold-lightning>
							</h4>
							<div v-if="showConfigurablePhases" class="mb-3 row">
								<label
									:for="formId('phases_0')"
									class="col-sm-4 col-form-label pt-0"
								>
									{{ $t("main.loadpointSettings.phasesConfigured.label") }}
								</label>
								<div class="col-sm-8 pe-0">
									<div class="form-check">
										<input
											:id="formId('phases_0')"
											v-model.number="selectedPhases"
											class="form-check-input"
											type="radio"
											:name="formId('phases')"
											:value="0"
											@change="changePhasesConfigured"
										/>
										<label class="form-check-label" :for="formId('phases_0')">
											{{
												$t(
													"main.loadpointSettings.phasesConfigured.phases_0"
												)
											}}
										</label>
									</div>
									<div class="form-check">
										<input
											:id="formId('phases_1')"
											v-model.number="selectedPhases"
											class="form-check-input"
											type="radio"
											:name="formId('phases')"
											:value="1"
											@change="changePhasesConfigured"
										/>
										<label class="form-check-label" :for="formId('phases_1')">
											{{
												$t(
													"main.loadpointSettings.phasesConfigured.phases_1"
												)
											}}
											<small>
												{{
													$t(
														"main.loadpointSettings.phasesConfigured.phases_1_hint",
														{ min: minPower1p, max: maxPower1p }
													)
												}}
											</small>
										</label>
									</div>
									<div class="form-check">
										<input
											:id="formId('phases_3')"
											v-model.number="selectedPhases"
											class="form-check-input"
											type="radio"
											:name="formId('phases')"
											:value="3"
											@change="changePhasesConfigured"
										/>
										<label class="form-check-label" :for="formId('phases_3')">
											{{
												$t(
													"main.loadpointSettings.phasesConfigured.phases_3"
												)
											}}
											<small>
												{{
													$t(
														"main.loadpointSettings.phasesConfigured.phases_3_hint",
														{ min: minPower3p, max: maxPower3p }
													)
												}}
											</small>
										</label>
									</div>
								</div>
							</div>

							<div v-if="$hiddenFeatures()" class="mb-3 row">
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

							<div v-if="$hiddenFeatures()" class="mb-3 row">
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
						<p class="small mt-3 text-muted mb-0">
							<strong class="text-evcc">
								{{ $t("main.loadpointSettings.disclaimerHint") }}
							</strong>
							{{ $t("main.loadpointSettings.disclaimerText") }}
						</p>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import "@h2d2/shopicons/es/bold/lightning";
import "@h2d2/shopicons/es/bold/car3";
import formatter from "../mixins/formatter";

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
	mixins: [formatter],
	props: {
		id: [String, Number],
		phasesConfigured: Number,
		phasesActive: Number,
		minSoc: Number,
		maxCurrent: Number,
		minCurrent: Number,
		title: String,
	},
	emits: ["phasesconfigured-updated", "maxcurrent-updated", "mincurrent-updated"],
	data: function () {
		return {
			selectedMaxCurrent: this.maxCurrent,
			selectedMinCurrent: this.minCurrent,
			selectedPhases: this.phasesConfigured,
		};
	},
	computed: {
		maxPower1p: function () {
			return this.fmtKw(this.maxCurrent * V);
		},
		minPower1p: function () {
			return this.fmtKw(this.minCurrent * V);
		},
		maxPower3p: function () {
			return this.fmtKw(this.maxCurrent * V * 3);
		},
		minPower3p: function () {
			return this.fmtKw(this.minCurrent * V * 3);
		},
		maxPower: function () {
			switch (this.phasesConfigured) {
				case PHASES_AUTO:
					return this.maxPower3p;
				case PHASES_3:
					return this.maxPower3p;
				case PHASES_1:
					return this.maxPower1p;
				default:
					return this.fmtKw(this.maxCurrent * V * this.phasesActive);
			}
		},
		minPower: function () {
			switch (this.phasesConfigured) {
				case PHASES_AUTO:
					return this.minPower1p;
				case PHASES_3:
					return this.minPower3p;
				case PHASES_1:
					return this.minPower1p;
				default:
					return this.fmtKw(this.minCurrent * V * this.phasesActive);
			}
		},
		showConfigurablePhases: function () {
			return [PHASES_AUTO, PHASES_3, PHASES_1].includes(this.phasesConfigured);
		},
		showCurrentSettings: function () {
			return this.$hiddenFeatures();
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
	methods: {
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
