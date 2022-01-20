<template>
	<div class="d-flex justify-content-between mb-2">
		<span class="d-flex flex-nowrap">
			<shopicon-regular-angledoublerightsmall
				v-if="!isSource"
				class="arrow"
				:class="{ 'arrow--active': active }"
			></shopicon-regular-angledoublerightsmall>
			<BatteryIcon v-if="isBattery" :soc="soc" />
			<component :is="`shopicon-regular-${icon}`" v-else></component>
			<shopicon-regular-angledoublerightsmall
				v-if="isSource"
				class="arrow"
				:class="{ 'arrow--active': active }"
			></shopicon-regular-angledoublerightsmall>
		</span>
		<span class="text-nowrap flex-grow-1 ms-3">{{ name }}</span>
		<span class="text-end text-nowrap ps-1 fw-bold"
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
	opacity: 0;
	transform: translateX(-20%);
	transition-property: opacity, transform;
	transition-duration: 0.75s, 5s;
	transition-timing-function: ease-in;
}
.arrow--active {
	opacity: 1;
	transform: translateX(0);
	transition-duration: 0.75s, 0.5s;
}
</style>
