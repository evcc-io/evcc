<template>
	<div class="d-flex justify-content-center gap-4 my-5 color-transition">
		<transition name="fade" mode="out-in">
			<component :is="leftIcon" :key="leftIconIndex" class="animated-icon"></component>
		</transition>
		<transition name="fade" mode="out-in">
			<component :is="centerIcon" :key="centerIconIndex" class="animated-icon"></component>
		</transition>
		<transition name="fade" mode="out-in">
			<component :is="rightIcon" :key="rightIconIndex" class="animated-icon"></component>
		</transition>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/batterythreequarters";
import "@h2d2/shopicons/es/regular/cablecharge";
import "@h2d2/shopicons/es/regular/car3";
import "@h2d2/shopicons/es/regular/car1";
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/heart";
import "@h2d2/shopicons/es/regular/home";
import "@h2d2/shopicons/es/regular/lightning";
import "@h2d2/shopicons/es/regular/moonstars";
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/receivepayment";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/clock";
import BatteryBoost from "../MaterialIcon/BatteryBoost.vue";
import SunUp from "../MaterialIcon/SunUp.vue";
import DynamicPrice from "../MaterialIcon/DynamicPrice.vue";
import { defineComponent, markRaw } from "vue";
import type { Timeout } from "@/types/evcc";

export default defineComponent({
	data() {
		return {
			leftIconIndex: 2,
			centerIconIndex: 6,
			rightIconIndex: 10,
			updatePosition: 0,
			icons: [
				"shopicon-regular-batterythreequarters",
				"shopicon-regular-cablecharge",
				"shopicon-regular-car3",
				"shopicon-regular-eco1",
				"shopicon-regular-heart",
				"shopicon-regular-home",
				"shopicon-regular-lightning",
				"shopicon-regular-moonstars",
				"shopicon-regular-powersupply",
				"shopicon-regular-receivepayment",
				"shopicon-regular-sun",
				"shopicon-regular-clock",
				"shopicon-regular-car1",
				markRaw(BatteryBoost),
				markRaw(SunUp),
				markRaw(DynamicPrice),
			],
			interval: null as Timeout,
		};
	},
	computed: {
		leftIcon() {
			return this.icons[this.leftIconIndex];
		},
		centerIcon() {
			return this.icons[this.centerIconIndex];
		},
		rightIcon() {
			return this.icons[this.rightIconIndex];
		},
	},
	mounted() {
		this.interval = setInterval(this.rotateIcons, 3000);
	},
	unmounted() {
		if (this.interval) {
			clearInterval(this.interval);
		}
	},
	methods: {
		getRandomIcon(excludeIndices: number[]) {
			const availableIndices = Array.from(Array(this.icons.length).keys()).filter(
				(i) => !excludeIndices.includes(i)
			);
			const randomIndex = Math.floor(Math.random() * availableIndices.length);
			return availableIndices[randomIndex];
		},
		rotateIcons() {
			const excludeIndices = [this.leftIconIndex, this.centerIconIndex, this.rightIconIndex];
			switch (this.updatePosition) {
				case 0:
					this.leftIconIndex = this.getRandomIcon(excludeIndices);
					break;
				case 1:
					this.centerIconIndex = this.getRandomIcon(excludeIndices);
					break;
				case 2:
					this.rightIconIndex = this.getRandomIcon(excludeIndices);
					break;
			}
			this.updatePosition = (this.updatePosition + Math.ceil(Math.random() * 2)) % 3;
		},
	},
});
</script>

<style scoped>
.animated-icon {
	width: 3.5rem !important;
	height: 3.5rem !important;
}

.fade-enter-active,
.fade-leave-active {
	transition: opacity 0.5s ease;
}

.fade-enter-from,
.fade-leave-to {
	opacity: 0;
}
</style>
