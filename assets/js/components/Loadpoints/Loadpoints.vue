<template>
	<div
		class="container container--loadpoint px-0 mb-md-2 d-flex flex-column justify-content-center"
		data-testid="loadpoints"
	>
		<div
			v-if="slides.length > 0"
			ref="carousel"
			class="carousel d-lg-flex flex-wrap"
			:class="[`carousel--${slides.length}`, { 'carousel--fullwidth': fullWidth }]"
		>
			<div
				v-for="(slide, index) in slides"
				:key="slide.key"
				class="flex-grow-1 mb-3 m-lg-0 p-lg-0"
				:class="{ 'loadpoint-unselected': !selected(index) }"
			>
				<PvCard
					v-if="slide.type === 'pv'"
					class="h-100 loadpoint"
					:pv="pv"
					:pvPower="pvPower"
					:pvEnergy="pvEnergy"
					:forecast="forecast"
					:experimental="experimental"
				/>
				<Loadpoint
					v-else
					v-bind="slide.loadpoint"
					data-testid="loadpoint"
					:vehicles="vehicles"
					:smartCostType="smartCostType"
					:smartCostAvailable="smartCostAvailable"
					:smartFeedInPriorityAvailable="smartFeedInPriorityAvailable"
					:tariffGrid="tariffGrid"
					:tariffCo2="tariffCo2"
					:tariffFeedIn="tariffFeedIn"
					:currency="currency"
					:multipleLoadpoints="multipleLoadpoints"
					:fullWidth="fullWidth"
					:gridConfigured="gridConfigured"
					:pvConfigured="pvConfigured"
					:batteryConfigured="batteryConfigured"
					:batterySoc="batterySoc"
					:batteryMode="batteryMode"
					:forecast="forecast"
					class="h-100"
					@click="goTo(slide.loadpoint.id)"
				/>
			</div>
		</div>
		<div v-if="slides.length > 1" class="d-flex d-lg-none justify-content-center flex-wrap">
			<button
				v-for="(slide, index) in slides"
				:key="`indicator-${slide.key}`"
				class="btn btn-sm btn-link p-0 mx-1 indicator d-flex justify-content-center align-items-center evcc-default-text"
				:class="{ 'indicator--selected': selected(index) }"
				@click="goTo(slide.type === 'pv' ? 'pv' : slide.loadpoint.id)"
			>
				<shopicon-regular-sun v-if="slide.type === 'pv'" class="indicator-icon" />
				<shopicon-filled-lightning
					v-else-if="isCharging(slide.loadpoint)"
					class="indicator-icon"
				></shopicon-filled-lightning>
				<shopicon-filled-circle
					v-else-if="slide.loadpoint.connected"
					class="indicator-icon"
				></shopicon-filled-circle>
				<shopicon-bold-circle v-else class="indicator-icon"></shopicon-bold-circle>
			</button>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/filled/circle";
import "@h2d2/shopicons/es/bold/circle";
import "@h2d2/shopicons/es/filled/lightning";
import "@h2d2/shopicons/es/regular/sun";

import Loadpoint from "./Loadpoint.vue";
import PvCard from "../Site/PvCard.vue";
import { defineComponent, type PropType } from "vue";
import type {
	UiLoadpoint,
	SMART_COST_TYPE,
	Timeout,
	Vehicle,
	BATTERY_MODE,
	Meter,
} from "@/types/evcc";

const PV_SLIDE_ID = "pv";

export default defineComponent({
	name: "Loadpoints",
	components: { Loadpoint, PvCard },
	props: {
		loadpoints: { type: Array as PropType<UiLoadpoint[]>, default: () => [] },
		pv: { type: Array as PropType<Meter[]>, default: () => [] },
		pvPower: { type: Number, default: 0 },
		pvEnergy: Number,
		vehicles: { type: Array as PropType<Vehicle[]> },
		smartCostType: String as PropType<SMART_COST_TYPE>,
		smartCostAvailable: Boolean,
		smartFeedInPriorityAvailable: Boolean,
		tariffGrid: Number,
		tariffCo2: Number,
		tariffFeedIn: Number,
		currency: String,
		selectedId: String,
		gridConfigured: Boolean,
		pvConfigured: Boolean,
		batteryConfigured: Boolean,
		batterySoc: Number,
		batteryMode: String as PropType<BATTERY_MODE>,
		experimental: Boolean,
		forecast: Object, // as PropType<Forecast>,
	},
	emits: ["id-changed"],
	data() {
		return {
			snapTimeout: null as Timeout,
			scrollTimeout: null as Timeout,
			highlightedIndex: 0,
			viewportHeight: 0 as number,
		};
	},
	computed: {
		slides() {
			const loadpointSlides = this.loadpoints.map((loadpoint) => ({
				type: "loadpoint" as const,
				key: `lp-${loadpoint.id}`,
				loadpoint,
			}));
			const pvSlides = this.pvConfigured ? [{ type: "pv" as const, key: "pv" as const }] : [];
			return [...loadpointSlides, ...pvSlides];
		},
		selectedIndex() {
			return this.indexById(this.selectedId);
		},
		multipleLoadpoints() {
			return this.loadpoints.length > 1;
		},
		fullWidth() {
			const tiles = this.slides.length;
			return (
				// breakpoint lg, tall screen, 2 loadpoints rows
				(tiles === 2 && this.viewportHeight >= 1450) ||
				// breakpoint lg, taller screen, 3 loadpoints rows
				(tiles === 3 && this.viewportHeight >= 1900)
			);
		},
	},
	watch: {
		selectedIndex(newIndex) {
			this.scrollTo(newIndex);
		},
	},
	mounted() {
		this.updateViewport();
		window.addEventListener("resize", this.updateViewport);

		if (this.selectedIndex > 0) {
			this.$refs["carousel"]?.scrollTo({ top: 0, left: this.left(this.selectedIndex) });
		}
		this.$refs["carousel"]?.addEventListener("scroll", this.handleCarouselScroll);
	},
	unmounted() {
		window.removeEventListener("resize", this.updateViewport);
		this.$refs["carousel"]?.removeEventListener("scroll", this.handleCarouselScroll);
	},
	methods: {
		indexById(id: string | undefined) {
			if (!id) return 0;
			if (this.pvConfigured && id === PV_SLIDE_ID) {
				return this.loadpoints.length;
			}
			const lpIndex = this.loadpoints.findIndex((lp) => lp.id === id);
			if (lpIndex < 0) return 0;
			return lpIndex;
		},
		idByIndex(index: number) {
			if (this.pvConfigured && index >= this.loadpoints.length) return PV_SLIDE_ID;
			return this.loadpoints[index]?.id;
		},
		handleCarouselScroll() {
			const { scrollLeft } = this.$refs["carousel"] as HTMLElement;
			const { offsetWidth } = this.$refs["carousel"]?.children[0] as HTMLElement;
			this.highlightedIndex = Math.round((scrollLeft - 7.5) / offsetWidth);

			// save scroll position to url if not changing for 2s
			if (this.scrollTimeout) {
				clearTimeout(this.scrollTimeout);
			}
			this.scrollTimeout = setTimeout(() => {
				if (this.highlightedIndex !== this.selectedIndex) {
					this.$emit("id-changed", this.idByIndex(this.highlightedIndex));
				}
			}, 2000);
		},
		goTo(id?: string) {
			this.$emit("id-changed", id);
		},
		isCharging(lp: UiLoadpoint) {
			return lp.charging && lp.chargePower > 0;
		},
		selected(index: number) {
			return this.highlightedIndex === index;
		},
		updateViewport() {
			this.viewportHeight = window.innerHeight;
		},
		left(index: number) {
			return (this.$refs["carousel"]?.children[0] as HTMLElement).offsetWidth * index;
		},
		scrollTo(index: number) {
			this.highlightedIndex = index;
			const $carousel = this.$refs["carousel"];
			if ($carousel) {
				$carousel.style.scrollSnapType = "none";
				$carousel?.scrollTo({ top: 0, left: this.left(index), behavior: "smooth" });
			}

			if (this.snapTimeout) {
				clearTimeout(this.snapTimeout);
			}
			this.snapTimeout = setTimeout(() => {
				if (this.$refs["carousel"]) {
					this.$refs["carousel"].style.scrollSnapType = "x mandatory";
				}
			}, 1000);
		},
	},
});
</script>
<style scoped>
@import "../../../css/breakpoints.css";

.container--loadpoint:not(:empty) {
	min-height: 300px;
}

@media (max-width: 991.98px) {
	.carousel {
		scroll-snap-type: x mandatory;
		overflow-x: scroll;
		padding-top: 0.75rem;
		display: flex;
		flex-wrap: nowrap !important;
		scrollbar-width: none; /* Firefox */
		-ms-overflow-style: none; /* IE 10+ */
	}
	.carousel::-webkit-scrollbar {
		display: none; /* Blink, Webkit */
	}
	.carousel > * {
		scroll-snap-align: center;
		min-width: 100%;
	}
	.indicator {
		width: 32px;
		height: 32px;
		opacity: 0.3;
		transition: opacity var(--evcc-transition-fast) ease-in;
	}
	.indicator--selected {
		opacity: 1;
	}
	.indicator-icon {
		width: 18px;
	}
	.loadpoint {
		opacity: 1;
		transform: scale(1);
		transition-property: opacity, transform;
		transition-duration: var(--evcc-transition-fast);
		transition-timing-function: ease-in;
	}
	.loadpoint-unselected {
		transform: scale(0.95);
		opacity: 0.5;
	}
}

/* show truncated tiles on breakpoint sm,md */
@media (--sm-to-lg) {
	.container--loadpoint {
		max-width: none;
	}
	.carousel > *:first-child {
		margin-left: calc((100vw - var(--slide-width)) / 2);
	}
	.carousel > *:last-child {
		margin-right: calc((100vw - var(--slide-width)) / 2);
	}
	/* fixes safari issue with end-side padding https://webplatform.news/issues/2019-08-07 */
	.carousel::after {
		content: "";
		padding-right: 0.02px;
	}
	.carousel > * {
		min-width: var(--slide-width);
	}
}

/* breakpoint sm */
@media (--sm-to-md) {
	.carousel {
		--slide-width: 540px;
	}
}

/* breakpoint md */
@media (--md-to-lg) {
	.carousel {
		--slide-width: 720px;
	}
}

/* breakpoint lg, 2-col grid */
@media (--lg-and-up) {
	.carousel {
		display: grid !important;
		grid-gap: 2rem;
		grid-template-columns: repeat(auto-fit, minmax(450px, 1fr));
	}
	/* breakpoint lg, full width override */
	.carousel--fullwidth {
		grid-gap: 4rem;
		grid-template-columns: 1fr;
	}
}
</style>
