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
			<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable" role="document">
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
							<h4 class="d-flex align-items-center mb-3 mt-0 text-evcc">
								{{ $t("main.loadpointSettings.vehicle") }}
								<shopicon-bold-car3 class="ms-2"></shopicon-bold-car3>
							</h4>
							<div class="mb-3 row">
								<label
									:for="formId('minsoc')"
									class="col-sm-4 col-form-label pt-0 pt-sm-1"
								>
									{{ $t("main.loadpointSettings.minSoC.label") }}
								</label>
								<div class="col-sm-8 pe-0">
									<select
										:id="formId('minsoc')"
										v-model.number="selectedMinSoC"
										class="form-select form-select-sm mb-2 w-50"
										@change="changeMinSoC"
									>
										<option
											v-for="soc in [
												0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50,
											]"
											:key="soc"
											:value="soc"
										>
											{{ soc ? `${soc}%` : "--" }}
										</option>
									</select>
									<small>
										{{
											$t("main.loadpointSettings.minSoC.description", [
												selectedMinSoC || "--",
											])
										}}
									</small>
								</div>
							</div>

							<h4 class="d-flex align-items-center mb-3 mt-4 text-evcc">
								{{ $t("main.loadpointSettings.currents") }}
								<shopicon-bold-lightning class="ms-1"></shopicon-bold-lightning>
							</h4>
							<div v-if="[0, 1, 3].includes(selectedPhases)" class="mb-3 row">
								<label
									:for="formId('phases_0')"
									class="col-sm-4 col-form-label pt-0"
								>
									{{ $t("main.loadpointSettings.phases1p3p.label") }}
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
											@change="changePhases1p3p"
										/>
										<label class="form-check-label" :for="formId('phases_0')">
											{{ $t("main.loadpointSettings.phases1p3p.phases_0") }}
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
											@change="changePhases1p3p"
										/>
										<label class="form-check-label" :for="formId('phases_1')">
											{{ $t("main.loadpointSettings.phases1p3p.phases_1") }}
											<small>
												{{
													$t(
														"main.loadpointSettings.phases1p3p.phases_1_hint",
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
											@change="changePhases1p3p"
										/>
										<label class="form-check-label" :for="formId('phases_3')">
											{{ $t("main.loadpointSettings.phases1p3p.phases_3") }}
											<small>
												{{
													$t(
														"main.loadpointSettings.phases1p3p.phases_3_hint",
														{ min: minPower3p, max: maxPower3p }
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
											v-for="{ value, name } in currentOptions(true, 16)"
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
											v-for="{ value, name } in currentOptions(false, 6)"
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

export default {
	name: "LoadpointSettingsModal",
	mixins: [formatter],
	props: {
		id: [String, Number],
		phases1p3p: Number,
		minSoC: Number,
		maxCurrent: Number,
		minCurrent: Number,
		title: String,
	},
	emits: ["phases1p3p-updated", "maxcurrent-updated", "mincurrent-updated", "minsoc-updated"],
	data: function () {
		return {
			selectedMaxCurrent: this.maxCurrent,
			selectedMinCurrent: this.minCurrent,
			selectedPhases: this.phases1p3p,
			selectedMinSoC: this.minSoC,
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
			return this.phases1p3p === 1 ? this.maxPower1p : this.maxPower3p;
		},
		minPower: function () {
			return this.phases1p3p === 3 ? this.minPower3p : this.minPower1p;
		},
	},
	watch: {
		maxCurrent: function (value) {
			this.selectedMaxCurrent = value;
		},
		minCurrent: function (value) {
			this.selectedMinCurrent = value;
		},
		phases1p3p: function (value) {
			this.selectedPhases = value;
		},
		minSoC: function (value) {
			this.selectedMinSoC = value;
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
		changePhases1p3p: function () {
			this.$emit("phases1p3p-updated", this.selectedPhases);
		},
		changeMinSoC: function () {
			this.$emit("minsoc-updated", this.selectedMinSoC);
		},
		currentOptions: function (max, defaultCurrent = 16) {
			const result = [];
			const toValue = max ? 32 : this.maxCurrent;
			const fromValue = max ? this.minCurrent : 6;
			for (let value = toValue; value >= fromValue; value--) {
				let name = `${value} A`;
				if (value === defaultCurrent) {
					name += ` (${this.$t("main.loadpointSettings.default")})`;
				}
				result.push({ value, name });
			}
			return result;
		},
	},
};
</script>
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
}
</style>
