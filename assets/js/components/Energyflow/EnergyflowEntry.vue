<template>
	<div class="d-flex justify-content-between">
		<span class="d-flex text-muted flex-nowrap">
			<shopicon-regular-angledoublerightsmall
				v-if="!isSource"
				:class="arrowClasses"
			></shopicon-regular-angledoublerightsmall>
			<BatteryIcon v-if="isBattery" :soc="soc" />
			<component :is="`shopicon-regular-${icon}`" v-else></component>
			<shopicon-regular-angledoublerightsmall
				v-if="isSource"
				:class="arrowClasses"
			></shopicon-regular-angledoublerightsmall>
		</span>
		<span class="text-nowrap flex-grow-1 ms-3">{{ name }}</span>
		<span class="text-end text-nowrap ps-1"
			><span v-if="hasSoC">{{ soc }}% / </span>{{ kw(power) }}</span
		>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/angledoublerightsmall";
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/home";
import "@h2d2/shopicons/es/regular/car3";
import BatteryIcon from "./BatteryIcon.vue";
import formatter from "../../mixins/formatter";

export default {
	name: "EnergyflowEntry",
	components: { BatteryIcon },
	mixins: [formatter],
	props: {
		name: { type: String },
		icon: { type: String },
		power: { type: Number },
		soc: { type: Number },
		type: { type: String }, // source, consumer
		valuesInKw: { type: Boolean },
	},
	computed: {
		active: function () {
			return this.power > 10;
		},
		isSource: function () {
			return this.type === "source";
		},
		isBattery: function () {
			return this.icon === "battery";
		},
		hasSoC: function () {
			return this.isBattery && !isNaN(this.soc);
		},
		arrowClasses: function () {
			return { arrow: true, "opacity-0": !this.active };
		},
	},
	methods: {
		kw: function (watt) {
			return this.fmtKw(watt, this.valuesInKw);
		},
	},
};
</script>
<style scoped>
.arrow {
	opacity: 100%;
	transition: opacity 1s ease-in;
}
</style>
