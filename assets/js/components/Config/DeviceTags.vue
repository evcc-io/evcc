<template>
	<div v-if="tags" class="list mb-3">
		<span
			v-for="(entry, index) in entries"
			:key="index"
			class=" entry"
		>
		<div class="label ">{{ $t(`config.deviceValue.${entry.name}`) }}</div>
		<div class="value">{{ fmtDeviceValue(entry) }}</div>
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
<style scoped>
.list {
	display: grid;
	grid-template-columns: 1fr;
	grid-gap: 0.5rem;
}
.entry {
	display: grid;
	grid-template-columns: 1fr 1fr;
	grid-gap: 1rem 1rem;
}
.value {
	display: block;
	font-weight: bold;
	color: var(--bs-primary);
	font-size: 14px;
	text-align: right;
}
.label {
	font-weight: normal;
	font-size: 14px;
}
</style>
