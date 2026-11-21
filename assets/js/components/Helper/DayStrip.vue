<template>
	<div class="d-flex flex-column gap-2 small">
		<div class="days d-flex" role="list" @mouseleave="hovered = null">
			<component
				:is="day.to ? 'router-link' : 'span'"
				v-for="(day, index) in days"
				:key="index"
				:to="day.to"
				role="listitem"
				:aria-label="ariaLabel(day)"
				class="day"
				:class="{
					hovered: hovered === index,
					faded: hovered === null ? day.state === 'none' : hovered !== index,
				}"
				@mouseenter="hovered = index"
				@focus="hovered = index"
				@blur="hovered = null"
			>
				<span class="box d-block" :class="`box--${day.state}`"></span>
			</component>
		</div>
		<div class="legend d-flex justify-content-between gap-2">
			<template v-if="hoveredDay">
				<span class="text-nowrap" :class="captionClass">{{ edgeLabel(hoveredDay) }}</span>
				<span class="text-nowrap" :class="captionClass">{{ hoveredDay.value }}</span>
			</template>
			<template v-else>
				<span class="text-nowrap">{{ edgeLabel(days.at(0)) }}</span>
				<span class="text-nowrap">{{ edgeLabel(days.at(-1)) }}</span>
			</template>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import type { RouteLocationRaw } from "vue-router";
import formatter from "@/mixins/formatter";

export type DayState = "ok" | "warning" | "none";

export interface Day {
	date: Date;
	state: DayState;
	/** Shown opposite the date on hover. */
	value?: string;
	to?: RouteLocationRaw;
}

export default defineComponent({
	name: "DayStrip",
	mixins: [formatter],
	props: {
		days: { type: Array as PropType<Day[]>, required: true },
	},
	data() {
		return { hovered: null as number | null };
	},
	computed: {
		hoveredDay(): Day | undefined {
			return this.hovered === null ? undefined : this.days[this.hovered];
		},
		captionClass(): string {
			return { ok: "text-primary", warning: "text-warning", none: "" }[
				this.hoveredDay?.state ?? "none"
			];
		},
	},
	methods: {
		edgeLabel(day?: Day): string {
			return day ? this.fmtDayMonthShort(day.date) : "";
		},
		ariaLabel(day: Day): string {
			return [this.edgeLabel(day), day.value].filter(Boolean).join(" ");
		},
	},
});
</script>

<style scoped>
.days {
	gap: 3px;
}
.day {
	flex: 1 1 0;
	min-width: 0;
	transition: opacity var(--evcc-transition-fast) ease-in;
}
.faded {
	opacity: 0.33;
}
.box {
	height: 20px;
	border-radius: 2px;
	transform-origin: bottom;
	transition: transform var(--evcc-transition-fast) ease-in;
}
.hovered .box {
	transform: scaleY(1.2);
}
.legend {
	color: var(--bs-gray-light);
}
.box--ok {
	background: var(--bs-primary);
}
.box--warning {
	background: var(--bs-warning);
}
.box--none {
	background: var(--bs-gray-light);
}
</style>
