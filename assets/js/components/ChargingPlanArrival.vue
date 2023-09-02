<template>
	<div class="mt-4 container">
		<div class="row">
			<div class="col-12 col-md-6">
				<div
					class="minsoc d-flex justify-content-between align-items-baseline mb-2 me-sm-4"
				>
					<label :for="formId('minsoc')" class="pt-1">
						{{ $t("main.loadpointSettings.minSoc.label") }}
					</label>
					<select
						:id="formId('minsoc')"
						v-model.number="selectedMinSoc"
						class="minSocSelect form-select mb-2"
						:disabled="!socBasedCharging"
						@change="changeMinSoc"
					>
						<option
							v-for="{ value, label } in minSocOptions"
							:key="value"
							:value="value"
						>
							{{ label }}
						</option>
					</select>
				</div>
			</div>
			<small class="col-12 col-md-6 ps-md-4">
				{{ $t("main.loadpointSettings.minSoc.description", [selectedMinSoc || "x"]) }}
			</small>
		</div>
		<div class="row">
			<div class="col-12 col-md-6">
				<div
					class="defaulttargetsoc d-flex justify-content-between align-items-baseline mb-2 me-sm-4"
				>
					<label :for="formId('defaulttargetsoc')" class="pt-1">
						{{ $t("main.loadpointSettings.defaultTargetSoc.label") }}
					</label>
					<select
						:id="formId('defaulttargetsoc')"
						v-model.number="selectedDefaultTargetSoc"
						class="minSocSelect form-select mb-2"
						:disabled="!socBasedCharging"
						@change="changeDefaultTargetSoc"
					>
						<option
							v-for="{ value, label } in defaultTargetSocOptions"
							:key="value"
							:value="value"
						>
							{{ label }}
						</option>
					</select>
				</div>
			</div>
			<small class="col-12 col-md-6 ps-md-4">
				{{ $t("main.loadpointSettings.defaultTargetSoc.description") }}
			</small>
		</div>
	</div>
</template>

<script>
export default {
	name: "ChargingPlanArrival",
	props: {
		id: [String, Number],
		minSoc: Number,
		vehicleName: String,
		socBasedCharging: Boolean,
	},
	emits: ["minsoc-updated", "defaulttargetsoc-updated"],
	data: function () {
		return {
			selectedMinSoc: this.minSoc,
			selectedDefaultTargetSoc: this.defaultTargetSoc,
		};
	},
	computed: {
		minSocOptions() {
			const options = [{ value: 0, label: "--" }];
			for (let i = 5; i <= 50; i += 5) {
				options.push({ value: i, label: `${i}%` });
			}
			return options;
		},
		defaultTargetSocOptions() {
			const options = [
				{ value: 0, label: this.$t("main.loadpointSettings.defaultTargetSoc.none") },
			];
			for (let i = 20; i <= 100; i += 5) {
				options.push({ value: i, label: `${i}%` });
			}
			return options;
		},
	},
	watch: {
		minSoc: function (value) {
			this.selectedMinSoc = value;
		},
		defaultTargetSoc: function (value) {
			this.selectedDefaultTargetSoc = value;
		},
	},
	methods: {
		formId: function (name) {
			return `chargingplan_${this.id}_${name}`;
		},
		changeMinSoc: function () {
			this.$emit("minsoc-updated", this.selectedMinSoc);
		},
		changeDefaultTargetSoc: function () {
			this.$emit("defaulttargetsoc-updated", this.selectedDefaultTargetSoc);
		},
	},
};
</script>

<style scoped>
.minsoc {
	max-width: 300px;
}
.minSocSelect {
	width: 100px;
}
</style>
