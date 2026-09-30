<template>
	<div class="humidity-bar" data-testid="humidity-bar">
		<div class="humidity-device-state mb-2" data-testid="humidity-device-state">
			{{ $t("main.loadpoint.humidity.device") }}:
			<strong>{{
				enabled ? $t("main.loadpoint.humidity.on") : $t("main.loadpoint.humidity.off")
			}}</strong>
		</div>
		<div
			class="progress humidity-track position-relative"
			role="group"
			:aria-label="$t('main.loadpoint.humidity.title')"
		>
			<div
				v-if="hasHumidity"
				class="progress-bar humidity-fill"
				role="progressbar"
				:aria-label="$t('main.loadpoint.humidity.current')"
				aria-valuemin="0"
				aria-valuemax="100"
				:aria-valuenow="humidityPosition"
				:style="{ width: `${humidityPosition}%` }"
				data-testid="humidity-fill"
			></div>
			<div
				class="humidity-target"
				:style="{ left: `${targetPosition}%` }"
				:aria-label="$t('main.loadpoint.humidity.target')"
				data-testid="humidity-target-marker"
			></div>
		</div>
		<div class="humidity-values d-flex justify-content-between mt-2">
			<div>
				<div class="humidity-label">{{ $t("main.loadpoint.humidity.current") }}</div>
				<strong data-testid="humidity-current-value">{{ currentValue }}</strong>
			</div>
			<div class="text-end">
				<div class="humidity-label">{{ $t("main.loadpoint.humidity.target") }}</div>
				<strong data-testid="humidity-target-value">{{ targetValue }}</strong>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";

const isValidHumidity = (value: number | null): value is number =>
	typeof value === "number" && Number.isFinite(value) && value >= 0 && value <= 100;

export default defineComponent({
	name: "HumidityBar",
	props: {
		humidity: { type: Number as PropType<number | null>, default: null },
		targetHumidity: { type: Number, required: true },
		enabled: Boolean,
	},
	computed: {
		hasHumidity(): boolean {
			return isValidHumidity(this.humidity);
		},
		humidityPosition(): number {
			return isValidHumidity(this.humidity) ? this.humidity : 0;
		},
		targetPosition(): number {
			return Math.max(0, Math.min(100, this.targetHumidity));
		},
		currentValue(): string {
			return isValidHumidity(this.humidity)
				? `${formatHumidity(this.humidity, this.$i18n?.locale)} %RH`
				: this.$t("main.loadpoint.humidity.unavailable");
		},
		targetValue(): string {
			return `${formatHumidity(this.targetHumidity, this.$i18n?.locale)} %RH`;
		},
	},
});

function formatHumidity(value: number, locale?: string): string {
	return new Intl.NumberFormat(locale, {
		minimumFractionDigits: 1,
		maximumFractionDigits: 1,
	}).format(value);
}
</script>

<style scoped>
.humidity-track {
	height: 0.75rem;
	overflow: visible;
}

.humidity-fill {
	background-color: var(--evcc-green);
}

.humidity-target {
	position: absolute;
	top: -0.25rem;
	bottom: -0.25rem;
	width: 2px;
	background: var(--evcc-default-text);
	border: 1px solid var(--evcc-box);
	transform: translateX(-50%);
}

.humidity-label {
	color: var(--evcc-gray);
	font-size: 0.875rem;
}
</style>
