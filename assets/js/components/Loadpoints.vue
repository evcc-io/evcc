<template>
	<div
		class="container container--loadpoint px-0 mb-md-2 d-flex flex-column justify-content-center"
	>
		<div ref="carousel" class="carousel d-lg-flex flex-wrap">
			<div
				v-for="(loadpoint, index) in loadpoints.slice(0, 2)"
				:key="index"
				class="flex-grow-1 mb-3 m-lg-0 p-lg-0"
			>
				<Loadpoint
					v-bind="loadpoint"
					:id="index + 1"
					:vehicles="vehicles"
					class="h-100"
					:class="{ 'loadpoint-unselected': !selected(index) }"
					@click="scrollTo(index)"
				/>
			</div>
			<div
				v-for="(group, groupIndex) in miniLoadpointGroups"
				:key="loadpoints + groupIndex"
				class="mini-loadpoints flex-grow-1 mb-3 m-lg-0 p-lg-0"
				:class="{
					[`mini-loadpoints--${miniLoadpointsPerPage}`]: true,
					'loadpoint-unselected': !selected(loadpoints.length + groupIndex),
				}"
				@click="scrollTo(loadpoints.length + groupIndex)"
			>
				<MiniLoadpoint
					v-for="(miniLoadpoint, index) in group"
					:id="(index % 2) + 1 /* TODO: real ids */"
					:key="index"
					v-bind="miniLoadpoint"
					class="h-100"
				/>
			</div>
		</div>
		<div v-if="loadpoints.length > 1" class="d-flex d-lg-none justify-content-center">
			<button
				v-for="index in loadpoints.length + 1"
				:key="index"
				class="btn btn-sm btn-link p-0 mx-1 indicator d-flex justify-content-center align-items-center evcc-default-text"
				:class="{ 'indicator--selected': selected(index) }"
				@click="scrollTo(index)"
			>
				<shopicon-filled-circle class="indicator-icon"></shopicon-filled-circle>
			</button>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/filled/circle";

import Loadpoint from "./Loadpoint.vue";
import MiniLoadpoint from "./MiniLoadpoint.vue";

const XXL_BREAKPOINT = 1400;

const mlp1 = {
	chargeCurrent: 2,
	chargedEnergy: 3261.87,
	chargePower: 400,
	chargerIcon: "cooler",
	maxCurrent: 6,
	minCurrent: 0,
	minSoc: 0,
	mode: "smart",
	phasesActive: 1,
	phasesConfigured: null,
	phasesEnabled: 1,
	title: "Air Conditioner",
};

const mlp2 = {
	chargeCurrent: 0,
	chargedEnergy: 361.87,
	chargePower: 0,
	chargerIcon: "scooter",
	maxCurrent: 6,
	minCurrent: 0,
	minSoc: 0,
	mode: "aus",
	phasesActive: 1,
	phasesConfigured: null,
	phasesEnabled: 1,
	title: "Scooter",
};

const mlp3 = {
	chargeCurrent: 7,
	chargedEnergy: 4421,
	chargePower: 1610,
	chargerIcon: "heater",
	maxCurrent: 16,
	minCurrent: 0,
	minSoc: 0,
	mode: "smart",
	phasesActive: 1,
	phasesConfigured: null,
	phasesEnabled: 1,
	title: "Heizstab",
};

const mlp4 = {
	chargeCurrent: 16,
	chargedEnergy: 412,
	chargePower: 882,
	chargerIcon: "generic",
	maxCurrent: 16,
	minCurrent: 0,
	minSoc: 0,
	mode: "an",
	phasesActive: 1,
	phasesConfigured: null,
	phasesEnabled: 1,
	title: "Kaffeemaschine",
};

const mlp5 = {
	chargeCurrent: 10,
	chargedEnergy: 122421,
	chargePower: 1610,
	chargerIcon: "waterheater",
	maxCurrent: 16,
	minCurrent: 0,
	minSoc: 0,
	mode: false,
	phasesActive: 1,
	phasesConfigured: null,
	phasesEnabled: 1,
	title: "Wärmepumpe",
};

export default {
	name: "Site",
	components: { Loadpoint, MiniLoadpoint },
	props: {
		loadpoints: Array,
		vehicles: Array,
	},
	data() {
		return { selectedIndex: 0, snapTimeout: null, miniLoadpointsPerPage: 2 };
	},
	computed: {
		miniLoadpointGroups() {
			// get real mini lps from api
			const loadpoints = [mlp1, mlp2, mlp3, mlp4, mlp5];

			const groups = [];
			const groupSize = this.miniLoadpointsPerPage;
			for (let i = 0; i < loadpoints.length; i += groupSize) {
				groups.push(loadpoints.slice(i, i + groupSize));
			}
			return groups;
		},
	},
	mounted() {
		window.addEventListener("resize", this.handleResize);
		this.handleResize();
		this.$refs.carousel.addEventListener("scroll", this.handleCarouselScroll, false);
	},
	unmounted() {
		window.removeEventListener("resize", this.handleResize);
		if (this.$refs.carousel) {
			this.$refs.carousel.removeEventListener("scroll", this.handleCarouselScroll);
		}
	},
	methods: {
		handleResize() {
			this.miniLoadpointsPerPage = window.innerWidth < XXL_BREAKPOINT ? 2 : 4;
		},
		handleCarouselScroll() {
			const { scrollLeft } = this.$refs.carousel;
			const { offsetWidth } = this.$refs.carousel.children[0];
			this.selectedIndex = Math.round((scrollLeft - 7.5) / offsetWidth);
		},
		selected(index) {
			return this.selectedIndex === index;
		},
		scrollTo(index) {
			if (this.selectedIndex === index) {
				return;
			}
			this.selectedIndex = index;
			const $carousel = this.$refs.carousel;
			const width = $carousel.children[0].offsetWidth;
			$carousel.style.scrollSnapType = "none";
			$carousel.scrollTo({ top: 0, left: 7.5 + width * index, behavior: "smooth" });

			clearTimeout(this.snapTimeout);
			this.snapTimeout = setTimeout(() => {
				this.$refs.carousel.style.scrollSnapType = "x mandatory";
			}, 1000);
		},
	},
};
</script>
<style scoped>
.container--loadpoint {
	min-height: 300px;
}

@media (max-width: 991.98px) {
	.carousel {
		scroll-snap-type: x mandatory;
		overflow-x: scroll;
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
@media (min-width: 576px) and (max-width: 991.98px) {
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
@media (min-width: 576px) and (max-width: 767.98px) {
	.carousel {
		--slide-width: 540px;
	}
}

/* breakpoint md */
@media (min-width: 768px) and (max-width: 991.98px) {
	.carousel {
		--slide-width: 720px;
	}
}

/* breakpoint lg */
@media (min-width: 992px) {
	.carousel {
		display: grid !important;
		grid-gap: 2rem;
		grid-template-columns: repeat(auto-fit, minmax(450px, 1fr));
	}
}

.mini-loadpoints {
	display: grid;
	grid-gap: 2rem;
}
.mini-loadpoints--2 {
	grid-template-rows: 1fr 1fr;
}
.mini-loadpoints--4 {
	grid-template-columns: 1fr 1fr;
	grid-template-rows: 1fr 1fr;
}
</style>
