<template>
	<div v-if="tags" class="d-flex mb-2 flex-wrap">
		<span
			v-for="(entry, index) in entries"
			:key="index"
			class="badge text-bg-secondary me-2 mb-2"
			:class="{
				'text-bg-secondary': !entry.error,
				'text-bg-danger': entry.error,
			}"
		>
			<strong>{{ $t(`config.deviceValue.${entry.name}`) }}:</strong>
			{{ fmtDeviceValue(entry) }}
		</span>
	</div>
</template>
<script>
import formatter from "../../mixins/formatter";

export default {
	name: "DeviceTags",
	props: {
		tags: Object,
	},
	mixins: [formatter],
	computed: {
		entries() {
			return Object.entries(this.tags).map(([name, { value, error }]) => {
				return { name, value, error };
			});
		},
	},
	methods: {
		fmtDeviceValue(entry) {
			const { name, value } = entry;
			switch (name) {
				case "power":
					return this.fmtKw(value);
				case "energy":
				case "capacity":
				case "chargedEnergy":
					return this.fmtKWh(value * 1e3);
				case "soc":
					return `${this.fmtNumber(value, 1)}%`;
				case "odometer":
				case "range":
					return `${this.fmtNumber(value, 0)} km`;
				case "phaseCurrents":
					return value.map((v) => this.fmtNumber(v, 0)).join(" ") + " A";
				case "phaseVoltages":
					return value.map((v) => this.fmtNumber(v, 0)).join(" ") + " V";
				case "phasePowers":
					return value.map((v) => this.fmtKw(v)).join(", ");
				case "chargeStatus":
					return value;
				case "socLimit":
					return `${this.fmtNumber(value)}%`;
			}
			return value;
		},
	},
};
</script>
<style scoped></style>
