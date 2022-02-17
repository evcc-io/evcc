<template>
	<div class="container px-0">
		<div ref="carousel" class="carousel d-lg-flex flex-wrap">
			<div
				v-for="(loadpoint, index) in loadpoints"
				:key="index"
				class="flex-grow-1 mb-3 me-lg-4 mx-lg-2 pb-lg-2"
			>
				<Loadpoint
					v-bind="loadpoint"
					:id="index"
					:class="{ 'loadpoint-unselected': !selected(index) }"
					@click="scrollTo(index)"
				/>
			</div>
		</div>
		<div v-if="loadpoints.length > 1" class="d-flex d-lg-none justify-content-center">
			<button
				v-for="(loadpoint, index) in loadpoints"
				:key="index"
				class="btn btn-sm btn-link p-0 mx-1 indicator d-flex justify-content-center align-items-center text-white"
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
import collector from "../mixins/collector";

export default {
	name: "Site",
	components: { Loadpoint },
	mixins: [collector],
	props: {
		loadpoints: Array,
	},
	data() {
		return { selectedIndex: 0 };
	},
	mounted() {
		this.$refs.carousel.addEventListener("scroll", this.handleCarouselScroll, false);
	},
	unmounted() {
		this.$refs.carousel.removeEventListener("scroll", this.handleCarouselScroll);
	},
	methods: {
		handleCarouselScroll() {
			const { offsetWidth, scrollLeft } = this.$refs.carousel;
			this.selectedIndex = Math.round(scrollLeft / offsetWidth);
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
			const width = $carousel.offsetWidth;
			$carousel.style.scrollSnapType = "none";
			$carousel.scrollTo({ top: 0, left: width * index, behavior: "smooth" });

			setTimeout(() => {
				this.$refs.carousel.style.scrollSnapType = "x mandatory";
			}, 500);
		},
	},
};
</script>
<style scoped>
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
		transition: opacity 0.2s ease-in;
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
		transition-duration: 0.2s;
		transition-timing-function: ease-in;
	}
	.loadpoint-unselected {
		transform: scale(0.95);
		opacity: 0.5;
	}
}

/* show truncated tiles on breakpoind sm,md */
@media (min-width: 576px) and (max-width: 991.98px) {
	.container {
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

/* breakpoind sm */
@media (min-width: 576px) and (max-width: 767.98px) {
	.carousel {
		--slide-width: 540px;
	}
}

/* breakpoind md */
@media (min-width: 768px) and (max-width: 991.98px) {
	.carousel {
		--slide-width: 720px;
	}
}
</style>
