<template>
	<div class="phase-hysteresis">
		<div
			ref="track"
			class="ph-track"
			@pointerdown="onPointerDown"
			@pointermove="onPointerMove"
			@pointerup="onPointerUp"
			@pointercancel="onPointerUp"
		>
			<div class="ph-line" />
			<div class="ph-band" :style="{ left: pct(downValue) + '%', width: bandWidth + '%' }" />
			<div v-if="!up" class="ph-ghost" :style="{ left: pct(defaultUp) + '%' }" />
			<div v-if="!down" class="ph-ghost" :style="{ left: pct(defaultDown) + '%' }" />
			<div
				class="ph-thumb"
				:class="{ 'ph-thumb-default': !down }"
				:style="{ left: pct(downValue) + '%' }"
				role="slider"
				tabindex="0"
				:aria-label="$t('config.loadpoint.phasesHysteresisDown')"
				:aria-valuemin="min"
				:aria-valuemax="max"
				:aria-valuenow="downValue"
				@keydown="onKeydown('down', $event)"
			>
				<span class="ph-bubble" :class="{ 'ph-bubble-default': !down }">{{
					fmtW(downValue, POWER_UNIT.AUTO)
				}}</span>
			</div>
			<div
				class="ph-thumb"
				:class="{ 'ph-thumb-default': !up }"
				:style="{ left: pct(upValue) + '%' }"
				role="slider"
				tabindex="0"
				:aria-label="$t('config.loadpoint.phasesHysteresisUp')"
				:aria-valuemin="min"
				:aria-valuemax="max"
				:aria-valuenow="upValue"
				@keydown="onKeydown('up', $event)"
			>
				<span class="ph-bubble" :class="{ 'ph-bubble-default': !up }">{{
					fmtW(upValue, POWER_UNIT.AUTO)
				}}</span>
			</div>
		</div>
		<div class="ph-ends d-flex justify-content-between evcc-gray">
			<span>{{ fmtW(min, POWER_UNIT.AUTO) }}</span>
			<span>{{ fmtW(max, POWER_UNIT.AUTO) }}</span>
		</div>
		<div class="d-flex flex-wrap justify-content-between gap-1 evcc-gray mt-1">
			<span class="ph-legend">
				▼&nbsp;{{ $t("config.loadpoint.phasesHysteresisDown") }}:
				<strong>{{
					down
						? fmtW(downValue, POWER_UNIT.AUTO)
						: $t("config.loadpoint.phasesHysteresisDefault")
				}}</strong>
			</span>
			<span class="ph-legend">
				▲&nbsp;{{ $t("config.loadpoint.phasesHysteresisUp") }}:
				<strong>{{
					up
						? fmtW(upValue, POWER_UNIT.AUTO)
						: $t("config.loadpoint.phasesHysteresisDefault")
				}}</strong>
			</span>
		</div>
		<div v-if="up || down" class="mt-1">
			<a href="#" @click.prevent="reset">{{
				$t("config.loadpoint.phasesHysteresisReset")
			}}</a>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import formatter from "@/mixins/formatter";
import { POWER_UNIT } from "@/mixins/formatter";

type DragTarget = "up" | "down";

export default defineComponent({
	name: "PhaseHysteresis",
	mixins: [formatter],
	props: {
		// thresholds in W, 0 = use derived default
		up: {
			type: Number,
			default: 0,
		},
		down: {
			type: Number,
			default: 0,
		},
		defaultUp: {
			type: Number,
			default: 0,
		},
		defaultDown: {
			type: Number,
			default: 0,
		},
		min: {
			type: Number,
			default: 440,
		},
		max: {
			type: Number,
			default: 11000,
		},
	},
	emits: ["update:up", "update:down"],
	data() {
		return {
			dragging: null as DragTarget | null,
		};
	},
	computed: {
		POWER_UNIT: () => POWER_UNIT,
		upValue(): number {
			return this.up || this.defaultUp || this.max;
		},
		downValue(): number {
			return this.down || this.defaultDown || this.min;
		},
		bandWidth(): number {
			return Math.max(0, this.pct(this.upValue) - this.pct(this.downValue));
		},
	},
	methods: {
		pct(value: number): number {
			const span = this.max - this.min;
			if (span <= 0) {
				return 0;
			}
			return Math.min(100, Math.max(0, ((value - this.min) / span) * 100));
		},
		valueFromClientX(clientX: number): number {
			const el = this.$refs["track"] as HTMLDivElement | undefined;
			if (!el) {
				return this.min;
			}
			const rect = el.getBoundingClientRect();
			if (rect.width <= 0) {
				return this.min;
			}
			const rel = Math.min(1, Math.max(0, (clientX - rect.left) / rect.width));
			return this.min + rel * (this.max - this.min);
		},
		thumbPosPx(value: number): number {
			const el = this.$refs["track"] as HTMLDivElement | undefined;
			if (!el) {
				return 0;
			}
			const rect = el.getBoundingClientRect();
			return rect.left + (this.pct(value) / 100) * rect.width;
		},
		onPointerDown(e: PointerEvent): void {
			const el = this.$refs["track"] as HTMLDivElement | undefined;
			if (!el) {
				return;
			}
			const downPx = this.thumbPosPx(this.downValue);
			const upPx = this.thumbPosPx(this.upValue);
			this.dragging =
				Math.abs(e.clientX - downPx) <= Math.abs(e.clientX - upPx) ? "down" : "up";
			el.setPointerCapture(e.pointerId);
			this.setValueFromX(this.dragging, e.clientX);
		},
		onPointerMove(e: PointerEvent): void {
			if (this.dragging) {
				this.setValueFromX(this.dragging, e.clientX);
			}
		},
		onPointerUp(): void {
			this.dragging = null;
		},
		setValueFromX(which: DragTarget, clientX: number): void {
			this.commit(which, Math.round(this.valueFromClientX(clientX) / 10) * 10);
		},
		onKeydown(which: DragTarget, e: KeyboardEvent): void {
			const step = e.shiftKey ? 100 : 10;
			let value = which === "up" ? this.upValue : this.downValue;
			switch (e.key) {
				case "ArrowLeft":
				case "ArrowDown":
					value -= step;
					break;
				case "ArrowRight":
				case "ArrowUp":
					value += step;
					break;
				case "Home":
					value = this.min;
					break;
				case "End":
					value = this.max;
					break;
				default:
					return;
			}
			e.preventDefault();
			this.commit(which, Math.round(value / 10) * 10);
		},
		commit(which: DragTarget, value: number): void {
			const clamped = Math.min(this.max, Math.max(this.min, value));
			if (which === "up") {
				const next = Math.max(clamped, this.downValue + 10);
				this.$emit("update:up", next);
			} else {
				const next = Math.min(clamped, this.upValue - 10);
				this.$emit("update:down", next);
			}
		},
		reset(): void {
			this.$emit("update:up", 0);
			this.$emit("update:down", 0);
		},
	},
});
</script>

<style scoped>
.phase-hysteresis {
	--ph-rail-h: 4px;
	--ph-thumb: 18px;
	padding: 0 0.5rem;
}
.ph-track {
	position: relative;
	height: 3rem;
	cursor: pointer;
	touch-action: none;
	user-select: none;
	-webkit-user-select: none;
}
.ph-line {
	position: absolute;
	left: 0;
	right: 0;
	bottom: 8px;
	height: var(--ph-rail-h);
	border-radius: 2px;
	background-color: var(--bs-secondary-bg);
}
.ph-band {
	position: absolute;
	bottom: 8px;
	height: var(--ph-rail-h);
	border-radius: 2px;
	background-color: color-mix(in srgb, var(--bs-primary) 35%, transparent);
	pointer-events: none;
}
.ph-ghost {
	position: absolute;
	bottom: 6px;
	width: 0;
	height: 16px;
	border-left: 1px dashed var(--bs-gray-dark);
	opacity: 0.5;
	pointer-events: none;
}
.ph-thumb {
	position: absolute;
	bottom: calc(8px + var(--ph-rail-h) / 2 - var(--ph-thumb) / 2);
	width: var(--ph-thumb);
	height: var(--ph-thumb);
	transform: translateX(-50%);
	border-radius: 50%;
	background-color: #fff;
	border: 2px solid var(--bs-primary);
	box-shadow: 0 1px 3px rgb(0 0 0 / 0.25);
	cursor: grab;
}
.ph-thumb:active {
	cursor: grabbing;
}
.ph-thumb:focus-visible {
	outline: 2px solid var(--bs-primary);
	outline-offset: 2px;
}
.ph-thumb-default {
	background-color: transparent;
	border-style: dashed;
	border-color: var(--bs-gray-dark);
}
.ph-bubble {
	position: absolute;
	bottom: calc(100% + 4px);
	left: 50%;
	transform: translateX(-50%);
	padding: 1px 6px;
	border-radius: 4px;
	background-color: var(--bs-primary);
	color: #fff;
	font-size: 0.75rem;
	font-weight: 600;
	white-space: nowrap;
	pointer-events: none;
}
.ph-bubble-default {
	background-color: var(--bs-gray-dark);
	color: #fff;
}
.ph-ends {
	font-size: 0.75rem;
	line-height: 1;
	margin-top: 2px;
}
.ph-legend {
	font-size: 0.875rem;
}
</style>
