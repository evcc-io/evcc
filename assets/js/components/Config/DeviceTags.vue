<template>
	<div v-if="tags" class="tags">
		<span
			v-for="(entry, index) in regularEntries"
			:key="index"
			:data-testid="`device-tag-${entry.name}`"
			class="d-flex gap-2 overflow-hidden text-truncate"
		>
			<div class="label overflow-hidden text-truncate flex-shrink-1 flex-grow-1">
				{{ $t(`config.deviceValue.${entry.name}`) }}
			</div>
			<div class="value overflow-hidden text-truncate" :class="valueClasses(entry)">
				{{ fmtDeviceValue(entry) }}
			</div>
		</span>
		<table v-if="hasPhaseEntries" class="table table-borderless table-sm my-0">
			<thead>
				<tr>
					<th></th>
					<th class="small evcc-gray text-end ps-2">L1</th>
					<th class="small evcc-gray text-end ps-2">L2</th>
					<th class="small evcc-gray text-end ps-2">L3</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				<tr
					v-for="(entry, index) in phaseEntries"
					:key="index"
					:data-testid="`device-tag-${entry.name}`"
				>
					<td class="text-truncate">
						{{ $t(`config.deviceValue.${entry.name}`) }}
					</td>
					<td
						v-for="(val, idx) in entry.value"
						:key="idx"
						class="value text-end tabular ps-2"
						:class="valueClasses(entry)"
					>
						{{ fmtPhaseValue(entry.name, val) }}
					</td>
					<td class="value unit-col ps-1" :class="valueClasses(entry)">
						{{ getPhaseUnit(entry.name) }}
					</td>
				</tr>
			</tbody>
		</table>
	</div>
</template>
<script>
import formatter, { POWER_UNIT } from "@/mixins/formatter";

const HIDDEN_TAGS = ["icon", "heating", "integratedDevice"];

const PHASE_TAGS = ["phaseCurrents", "phaseVoltages", "phasePowers"];

export default {
	name: "DeviceTags",
	mixins: [formatter],
	props: {
		tags: Object,
	},
	computed: {
		regularEntries() {
			return Object.entries(this.tags)
				.filter(([name]) => !HIDDEN_TAGS.includes(name) && !PHASE_TAGS.includes(name))
				.map(([name, { value, error, warning, muted, options }]) => {
					return { name, value, error, warning, muted, options };
				});
		},
		phaseEntries() {
			return Object.entries(this.tags)
				.filter(([name]) => PHASE_TAGS.includes(name))
				.map(([name, { value, error, warning, muted, options }]) => {
					return { name, value, error, warning, muted, options };
				});
		},
		hasPhaseEntries() {
			return this.phaseEntries.length > 0;
		},
	},
	methods: {
		valueClasses(entry) {
			return {
				"value--error": !!entry.error,
				"value--warning": entry.warning,
				"value--muted": entry.muted || entry.value === false,
			};
		},
		fmtDeviceValue(entry) {
			const { name, value, options = {} } = entry;
			if (value === null || value === undefined) {
				return "";
			}
			switch (name) {
				case "power":
				case "solarForecast":
				case "hemsActiveLimit":
					return this.fmtW(value);
				case "energy":
				case "capacity":
				case "chargedEnergy":
					return this.fmtWh(value * 1e3);
				case "soc":
				case "vehicleLimitSoc":
					return this.fmtPercentage(value, 1);
				case "temp":
				case "heaterTempLimit":
					return this.fmtTemperature(value);
				case "odometer":
				case "range":
					return `${this.fmtNumber(value, 0)} km`;
				case "chargeStatus":
					return value ? this.$t(`config.deviceValue.chargeStatus${value}`) : "-";
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
				case "phases1p3p":
				case "singlePhase":
				case "enabled":
				case "configured":
				case "dimmed":
					return value
						? this.$t("config.deviceValue.yes")
						: this.$t("config.deviceValue.no");
				case "hemsType":
					return this.$t(`config.deviceValueHemsType.${value}`);
			}
			return value;
		},
		fmtPhaseValue(name, value) {
			if (value === null || value === undefined) {
				return "–";
			}
			switch (name) {
				case "phaseCurrents":
					return this.fmtNumber(value, 1);
				case "phaseVoltages":
					return this.fmtNumber(value, 0);
				case "phasePowers":
					return this.fmtW(value, POWER_UNIT.KW, false);
			}
			return value;
		},
		getPhaseUnit(name) {
			switch (name) {
				case "phaseCurrents":
					return "A";
				case "phaseVoltages":
					return "V";
				case "phasePowers":
					return "kW";
			}
			return "";
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
table th,
table td {
	padding: 0 0 0.5rem 0;
	white-space: nowrap;
}
.unit-col {
	width: 0.1%;
}
</style>
