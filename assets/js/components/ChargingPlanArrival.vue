<template>
	<div class="mt-4 container">
		<div class="row">
			<div class="col-6 col-lg-3 col-form-label">
				<label :for="formId('minsoc')">
					{{ $t("main.loadpointSettings.minSoc.label") }}
				</label>
			</div>
			<div class="col-6 col-lg-3">
				<select
					:id="formId('minsoc')"
					v-model.number="selectedMinSoc"
					class="form-select mb-2"
					:disabled="!socBasedCharging"
					@change="changeMinSoc"
				>
					<option v-for="soc in minSocOptions" :key="soc.value" :value="soc.value">
						{{ soc.name }}
					</option>
				</select>
			</div>
			<small class="col-12 col-lg-6 ps-lg-4 col-form-label mb-4">
				{{ $t("main.loadpointSettings.minSoc.description", [selectedMinSoc || "x"]) }}
			</small>
		</div>
		<div class="row">
			<div class="col-6 col-lg-3 col-form-label">
				<label :for="formId('limitsoc')">
					{{ $t("main.loadpointSettings.limitSoc.label") }}
				</label>
			</div>
			<div class="col-6 col-lg-3">
				<select
					:id="formId('limitsoc')"
					v-model.number="selectedLimitSoc"
					class="form-select mb-2"
					:disabled="!socBasedCharging"
					@change="changeLimitSoc"
				>
					<option v-for="soc in limitSocOptions" :key="soc.value" :value="soc.value">
						{{ soc.name }}
					</option>
				</select>
			</div>
			<small class="col-12 col-lg-6 ps-lg-4 col-form-label mb-4">
				{{ $t("main.loadpointSettings.limitSoc.description") }}
			</small>
		</div>
	</div>
	<div v-if="!socBasedCharging" class="mx-2 small text-muted">
		<strong class="text-evcc">
			{{ $t("main.loadpointSettings.disclaimerHint") }}
		</strong>
		{{ $t("main.loadpointSettings.onlyForSocBasedCharging") }}
	</div>
</template>

<script>
import { distanceUnit } from "../units";
import formatter from "../mixins/formatter";

export default {
	name: "ChargingPlanArrival",
	mixins: [formatter],
	props: {
		id: [String, Number],
		minSoc: { type: Number, default: 0 },
		limitSoc: { type: Number, default: 0 },
		vehicleName: String,
		socBasedCharging: Boolean,
		rangePerSoc: Number,
	},
	emits: ["minsoc-updated", "limitsoc-updated"],
	data: function () {
		return { selectedMinSoc: this.minSoc, selectedLimitSoc: this.limitSoc };
	},
	computed: {
		minSocOptions() {
			// a list of entries from 0 to 95 with a step of 5
			return Array.from(Array(20).keys())
				.map((i) => i * 5)
				.map(this.socOption);
		},
		limitSocOptions() {
			// a list of entries from 0 to 100 with a step of 5
			return Array.from(Array(21).keys())
				.map((i) => i * 5)
				.map(this.socOption);
		},
	},
	watch: {
		minSoc: function (value) {
			this.selectedMinSoc = value;
		},
		limitSoc: function (value) {
			this.selectedLimitSoc = value;
		},
	},
	methods: {
		socOption: function (soc) {
			return {
				value: soc,
				name: soc === 0 ? "---" : this.fmtSocOption(soc, this.rangePerSoc, distanceUnit()),
			};
		},
		formId: function (name) {
			return `chargingplan_${this.id}_${name}`;
		},
		changeMinSoc: function () {
			this.$emit("minsoc-updated", this.selectedMinSoc);
		},
		changeLimitSoc: function () {
			this.$emit("limitsoc-updated", this.selectedLimitSoc);
		},
	},
};
</script>
