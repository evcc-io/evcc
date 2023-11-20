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
						<option v-for="soc in minSocOptions" :key="soc.value" :value="soc.value">
							{{ soc.name }}
						</option>
					</select>
				</div>
			</div>
			<small class="col-12 col-md-6 ps-md-4">
				{{ $t("main.loadpointSettings.minSoc.description", [selectedMinSoc || "x"]) }}
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
	emits: ["minsoc-updated"],
	data: function () {
		return { selectedMinSoc: this.minSoc };
	},
	computed: {
		minSocOptions() {
			// a list of entries from 0 to 95 with a step of 5
			return Array.from(Array(20).keys())
				.map((i) => i * 5)
				.map((soc) => {
					return { value: soc, name: soc === 0 ? "--" : `${soc}%` };
				});
		},
	},
	watch: {
		minSoc: function (value) {
			this.selectedMinSoc = value;
		},
	},
	methods: {
		formId: function (name) {
			return `chargingplan_${this.id}_${name}`;
		},
		changeMinSoc: function () {
			this.$emit("minsoc-updated", this.selectedMinSoc);
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
