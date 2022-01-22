<template>
	<div class="details d-flex align-items-start mb-3">
		<div>
			<div class="mb-2 label">
				{{ $t("main.loadpointDetails.power") }}
				<div
					v-if="chargePower && activePhases"
					v-tooltip="{ content: phaseTooltip }"
					class="badge rounded-pill bg-secondary text-light cursor-pointer d-inline-flex align-items-center"
					tabindex="0"
				>
					<div>{{ activePhases }}P</div>
					<WaitingDots
						v-if="phaseTimerVisible"
						:direction="phaseAction === 'scale1p' ? 'down' : 'up'"
					/>
				</div>
			</div>
			<h3 class="value">
				{{ fmt(chargePower) }}
				<small class="text-muted">{{ fmtUnit(chargePower) }}W</small
				><small
					v-if="pvTimerVisible"
					v-tooltip="{
						content: $t(`main.loadpointDetails.tooltip.pv.${pvAction}`, {
							remaining: fmtShortDuration(pvRemainingInterpolated, true),
						}),
					}"
					class="text-muted cursor-pointer d-inline-block align-bottom"
					style="margin-bottom: 0.1em"
					tabindex="0"
				>
					<WaitingDots :direction="pvAction === 'disable' ? 'down' : 'up'" />
				</small>
			</h3>
		</div>

		<div>
			<div class="mb-2 label">{{ $t("main.loadpointDetails.charged") }}</div>
			<h3 class="value">
				{{ fmt(chargedEnergy) }}
				<small class="text-muted">{{ fmtUnit(chargedEnergy) }}Wh</small>
			</h3>
		</div>

		<div v-if="vehiclePresent">
			<div class="mb-2 label">{{ $t("main.loadpointDetails.remaining") }}</div>
			<h3 class="value">
				{{ fmtShortDuration(chargeRemainingDurationInterpolated) }}
				<small class="text-muted">{{
					fmtShortDurationUnit(chargeRemainingDurationInterpolated, true)
				}}</small>
			</h3>
		</div>

		<div v-else>
			<div class="mb-2 label">{{ $t("main.loadpointDetails.duration") }}</div>
			<h3 class="value">
				{{ fmtShortDuration(chargeDurationInterpolated) }}
				<small class="text-muted">{{
					fmtShortDurationUnit(chargeDurationInterpolated)
				}}</small>
			</h3>
		</div>
	</div>
</template>

<script>
import WaitingDots from "./WaitingDots";
import formatter from "../mixins/formatter";

export default {
	name: "LoadpointDetails",
	components: { WaitingDots },
	mixins: [formatter],
	props: {
		chargedEnergy: Number,
		chargeDuration: Number,
		chargeRemainingDuration: Number,
		chargePower: Number,
		climater: String,
		vehiclePresent: Boolean,
		vehicleRange: Number,
		activePhases: Number,
		phaseRemaining: Number,
		phaseAction: String,
		pvRemaining: Number,
		pvAction: String,
		charging: Boolean,
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
		phaseTimerVisible() {
			if (this.phaseTimerActive && !this.pvTimerActive) {
				return true;
			}
			if (this.phaseTimerActive && this.pvTimerActive) {
				return this.phaseRemainingInterpolated < this.pvRemainingInterpolated; // only show next timer
			}
			return false;
		},
		pvTimerVisible() {
			if (this.pvTimerActive && !this.phaseTimerActive) {
				return true;
			}
			if (this.pvTimerActive && this.phaseTimerActive) {
				return this.pvRemainingInterpolated < this.phaseRemainingInterpolated; // only show next timer
			}
			return false;
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
	},
};
</script>

<style scoped>
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
.label {
	text-transform: uppercase;
	color: var(--bs-gray-medium);
	font-size: 16px;
}
</style>
