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
				/>
			</div>
		</div>
		<div class="d-flex d-lg-none justify-content-center">
			<button
				v-for="(loadpoint, index) in loadpoints"
				:key="index"
				class="btn btn-sm btn-link p-0 mx-1 indicator d-flex justify-content-center align-items-center text-white"
				@click="scrollTo(index)"
			>
				<!--<shopicon-bold-lightning
					v-if="loadpoint.charging && selected(index)"
					size="s"
				></shopicon-bold-lightning>
				<shopicon-light-lightning
					v-if="loadpoint.charging && !selected(index)"
					size="s"
				></shopicon-light-lightning>-->
				<div
					class="indicator--dot rounded-circle"
					:class="{ 'bg-white': selected(index) }"
				></div>
			</button>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/light/lightning";
import "@h2d2/shopicons/es/bold/lightning";

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
	}
	.indicator--dot {
		width: 10px;
		height: 10px;
		border: 1px solid var(--bs-white);
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
		opacity: 0.8;
	}
}
</style>
