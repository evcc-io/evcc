<template>
	<div class="d-flex justify-content-between mb-2 entry" :class="{ 'evcc-gray': !active }">
		<span class="d-flex flex-nowrap">
			<BatteryIcon v-if="isBattery" :soc="soc" />
			<VehicleIcon v-else-if="isVehicle" :names="vehicleIcons" />
			<component :is="`shopicon-regular-${icon}`" v-else></component>
		</span>
		<span class="text-nowrap flex-grow-1 ms-3">{{ name }}</span>
		<span class="text-end text-nowrap ps-1 fw-bold">
			<span v-if="price && active && showPrice()">
				{{ fmtPricePerKWh(price, currency, true) }} /
			</span>
			<span v-if="co2 && active && showCo2()">
				{{ fmtPricePerKWh(co2, currency, true) }} /
			</span>
			<span v-if="hasSoc">{{ soc }}% / </span>
			<AnimatedNumber :to="power" :format="kw" />
		</span>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/home";
import BatteryIcon from "./BatteryIcon.vue";
import formatter from "../../mixins/formatter";
import AnimatedNumber from "../AnimatedNumber.vue";
import VehicleIcon from "../VehicleIcon";
import { showGridPrice, showGridCo2 } from "../../gridDetails";

export default {
	name: "EnergyflowEntry",
	components: { BatteryIcon, AnimatedNumber, VehicleIcon },
	mixins: [formatter],
	props: {
		name: { type: String },
		icon: { type: String },
		power: { type: Number },
		soc: { type: Number },
		price: { type: Number },
		valuesInKw: { type: Boolean },
		vehicleIcons: { type: Array },
		currency: { type: String },
	},
	computed: {
		active: function () {
			return this.power > 10;
		},
		isBattery: function () {
			return this.icon === "battery";
		},
		isVehicle: function () {
			return this.icon === "vehicle";
		},
		hasSoc: function () {
			return this.isBattery && !isNaN(this.soc);
		},
	},
	methods: {
		showPrice() {
			return showGridPrice();
		},
		showCo2() {
			return showGridCo2();
		},
		kw: function (watt) {
			return this.fmtKw(watt, this.valuesInKw);
		},
	},
};
</script>
<style scoped>
.entry {
	transition: color var(--evcc-transition-medium) linear;
}
</style>
