<template>
	<div class="loadpoint bg-white px-4 px-sm-5 py-4">
		<div class="d-flex justify-content-between align-items-center mb-4 flex-wrap">
			<h3 class="mb-2 me-2">
				{{ title || $t("main.loadpoint.fallbackName") }}
			</h3>
			<Mode class="mb-2" :mode="mode" @updated="setTargetMode" />
		</div>

		<div
			v-if="remoteDisabled"
			class="alert alert-warning my-4 py-2"
			:class="`${remoteDisabled === 'hard' ? 'alert-danger' : 'alert-warning'}`"
			role="alert"
		>
			{{
				$t(
					remoteDisabled === "hard"
						? "main.loadpoint.remoteDisabledHard"
						: "main.loadpoint.remoteDisabledSoft",
					{ source: remoteDisabledSource }
				)
			}}
		</div>

		<div class="details d-flex align-items-start mb-3">
			<div class="d-flex align-items-center">
				<div>
					<LabelAndValue
						:label="$t('main.loadpointDetails.power')"
						:value="fmtKw(chargePower)"
					/>
					<Phases
						v-bind="phasesProps"
						class="opacity-transiton"
						:class="`opacity-${charging ? '100' : '0'}`"
					/>
				</div>
				<shopicon-regular-lightning
					class="text-evcc opacity-transiton"
					:class="`opacity-${charging ? '100' : '0'}`"
					size="m"
				></shopicon-regular-lightning>
			</div>
			<LabelAndValue
				:label="$t('main.loadpointDetails.charged')"
				:value="fmtKw(chargedEnergy)"
			/>
			<LabelAndValue
				v-if="chargeRemainingDurationInterpolated"
				:label="$t('main.loadpointDetails.remaining')"
				:value="`
					${fmtShortDuration(chargeRemainingDurationInterpolated)}
					${fmtShortDurationUnit(chargeRemainingDurationInterpolated, true)}`"
			/>
			<LabelAndValue
				v-else
				:label="$t('main.loadpointDetails.duration')"
				:value="`
					${fmtShortDuration(chargeDurationInterpolated)}
					${fmtShortDurationUnit(chargeDurationInterpolated)}`"
			/>
		</div>

		<Vehicle
			v-bind="vehicle"
			@target-soc-updated="setTargetSoC"
			@target-time-updated="setTargetTime"
			@target-time-removed="removeTargetTime"
		/>
	</div>
</template>

<script>
import api from "../api";
import Mode from "./Mode";
import Vehicle from "./Vehicle";
import Phases from "./Phases";
import LabelAndValue from "./LabelAndValue";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";
import "@h2d2/shopicons/es/regular/lightning";

export default {
	name: "Loadpoint",
	components: { Mode, Vehicle, Phases, LabelAndValue },
	mixins: [formatter, collector],
	props: {
		id: Number,
		single: Boolean,

		// main
		title: String,
		mode: String,
		targetSoC: Number,
		remoteDisabled: Boolean,
		remoteDisabledSource: String,
		chargeDuration: Number,
		charging: Boolean,

		// vehicle
		connected: Boolean,
		// charging: Boolean,
		enabled: Boolean,
		vehicleTitle: String,
		vehicleSoC: Number,
		vehiclePresent: Boolean,
		vehicleRange: Number,
		minSoC: Number,
		targetTime: String,
		targetTimeActive: Boolean,
		targetTimeHourSuggestion: Number,

		// details
		chargePower: Number,
		chargedEnergy: Number,
		// chargeDuration: Number,
		climater: String,
		chargeRemainingDuration: Number,

		// other information
		phases: Number,
		minCurrent: Number,
		maxCurrent: Number,
		activePhases: Number,
		chargeCurrent: Number,
		vehicleCapacity: Number,
		connectedDuration: Number,
		chargeCurrents: Array,
		chargeConfigured: Boolean,
		chargeRemainingEnergy: Number,
		phaseAction: String,
		phaseRemaining: Number,
		pvRemaining: Number,
		pvAction: String,
	},
	data() {
		return {
			tickerHandler: null,
			phaseRemainingInterpolated: this.phaseRemaining,
			pvRemainingInterpolated: this.pvRemaining,
			chargeDurationInterpolated: this.chargeDuration,
			chargeRemainingDurationInterpolated: this.chargeRemainingDuration,
		};
	},
	computed: {
		phaseTooltip() {
			if (["scale1p", "scale3p"].includes(this.phaseAction)) {
				return this.$t(`main.loadpointDetails.tooltip.phases.${this.phaseAction}`, {
					remaining: this.fmtShortDuration(this.phaseRemainingInterpolated, true),
				});
			}
			return this.$t(`main.loadpointDetails.tooltip.phases.charge${this.activePhases}p`);
		},
		phaseTimerActive() {
			return (
				this.phaseRemainingInterpolated > 0 &&
				["scale1p", "scale3p"].includes(this.phaseAction)
			);
		},
		pvTimerActive() {
			return (
				this.pvRemainingInterpolated > 0 && ["enable", "disable"].includes(this.pvAction)
			);
		},
		phasesProps: function () {
			return this.collectProps(Phases);
		},
		vehicle: function () {
			return this.collectProps(Vehicle);
		},
	},
	watch: {
		phaseRemaining() {
			this.phaseRemainingInterpolated = this.phaseRemaining;
		},
		pvRemaining() {
			this.pvRemainingInterpolated = this.pvRemaining;
		},
		chargeDuration() {
			this.chargeDurationInterpolated = this.chargeDuration;
		},
		chargeRemainingDuration() {
			this.chargeDurationInterpolated = this.chargeRemainingDuration;
		},
	},
	mounted() {
		this.tickerHandler = setInterval(this.tick, 1000);
	},
	destroyed() {
		clearInterval(this.tickerHandler);
	},
	methods: {
		tick() {
			if (this.phaseRemainingInterpolated > 0) {
				this.phaseRemainingInterpolated--;
			}
			if (this.pvRemainingInterpolated > 0) {
				this.pvRemainingInterpolated--;
			}
			if (this.chargeDurationInterpolated > 0 && this.charging) {
				this.chargeDurationInterpolated++;
			}
			if (this.chargeRemainingDurationInterpolated > 0 && this.charging) {
				this.chargeRemainingDurationInterpolated--;
			}
		},
		apiPath: function (func) {
			return "loadpoints/" + this.id + "/" + func;
		},
		setTargetMode: function (mode) {
			api.post(this.apiPath("mode") + "/" + mode);
		},
		setTargetSoC: function (soc) {
			api.post(this.apiPath("targetsoc") + "/" + soc);
		},
		setTargetTime: function (date) {
			const formattedDate = `${this.fmtDayString(date)}T${this.fmtTimeString(date)}:00`;
			api.post(this.apiPath("targetcharge") + "/" + this.targetSoC + "/" + formattedDate);
		},
		removeTargetTime: function () {
			api.delete(this.apiPath("targetcharge"));
		},
	},
};
</script>

<style scoped>
.loadpoint {
	border-radius: 20px;
	color: var(--bs-gray-dark);
}

.details > div {
	flex-grow: 1;
	flex-basis: 0;
}
.details > div:nth-child(2) {
	text-align: center;
}
.details > div:nth-child(3) {
	text-align: right;
}
.opacity-transiton {
	transition: opacity 0.7w5s ease-in;
}
</style>
