<template>
	<div v-if="tags" class="tags">
		<span
			v-for="(entry, index) in entries"
			:key="index"
			:data-testid="`device-tag-${entry.name}`"
			class="d-flex gap-2 overflow-hidden text-truncate"
		>
			<div class="label overflow-hidden text-truncate flex-shrink-1 flex-grow-1">
				{{ $t(`config.deviceValue.${entry.name}`) }}
			</div>
			<div
				class="value overflow-hidden"
				:class="{
					'value--error': !!entry.error,
					'value--warning': entry.warning,
					'value--muted': entry.muted || entry.value === false,
					'text-truncate': allowTruncate(entry.name),
					'flex-shrink-0': !allowTruncate(entry.name),
				}"
			>
				{{ fmtDeviceValue(entry) }}
			</div>
		</span>
	</div>
</template>
<script>
import formatter, { POWER_UNIT } from "../../mixins/formatter";

const NO_TRUNCATE = ["phasePowers", "phaseVoltages", "phaseCurrents"];

export default {
	name: "DeviceTags",
	mixins: [formatter],
	props: {
		tags: Object,
	},
	computed: {
		entries() {
			return Object.entries(this.tags).map(
				([name, { value, error, warning, muted, options }]) => {
					return { name, value, error, warning, muted, options };
				}
			);
		},
	},
	methods: {
		fmtDeviceValue(entry) {
			const { name, value, options = {} } = entry;
			if (value === null || value === undefined) {
				return "";
			}
			switch (name) {
				case "power":
					return this.fmtW(value);
				case "energy":
				case "capacity":
				case "chargedEnergy":
					return this.fmtWh(value * 1e3);
				case "soc":
				case "socLimit":
					return this.fmtPercentage(value, 1);
				case "temp":
					return this.fmtTemperature(value);
				case "odometer":
				case "range":
					return `${this.fmtNumber(value, 0)} km`;
				case "phaseCurrents":
					return value.map((v) => this.fmtNumber(v, 1)).join(" · ") + " A";
				case "phaseVoltages":
					return value.map((v) => this.fmtNumber(v, 0)).join(" · ") + " V";
				case "phasePowers":
					return value.map((v) => this.fmtW(v, POWER_UNIT.KW, false)).join(" · ") + " kW";
				case "chargeStatus":
					return this.$t(`config.deviceValue.chargeStatus${value}`);
				case "gridPrice":
				case "feedinPrice":
					return this.fmtPricePerKWh(value, options.currency, true);
				case "co2":
					return this.fmtCo2Short(value);
				case "powerRange":
					return `${this.fmtW(value[0])} / ${this.fmtW(value[1])}`;
				case "currentRange":
					return `${this.fmtNumber(value[0], 1)} A / ${this.fmtNumber(value[1], 1)} A`;
				case "controllable":
				case "configured":
					return value
						? this.$t("config.deviceValue.yes")
						: this.$t("config.deviceValue.no");
			}
			return value;
		},
		allowTruncate(name) {
			return !NO_TRUNCATE.includes(name);
		},
	},
};
</script>
<style scoped>
.tags {
	display: grid;
	grid-template-columns: 1fr;
	grid-gap: 0.5rem;
}
.label {
	min-width: 4rem;
}
.value {
	font-weight: bold;
	color: var(--bs-primary);
}
.value:empty::after {
	color: var(--evcc-gray);
	content: "–";
}
.value--error {
	color: var(--bs-danger);
}
.value--warning {
	color: var(--bs-warning);
}
.value--muted {
	color: var(--evcc-gray) !important;
}
</style>
