<template>
	<component :is="singleIcon" v-if="single" :class="`icon icon--${size}`"></component>
	<MultiIcon v-else :count="count" :size="size"></MultiIcon>
</template>

<script>
import "@h2d2/shopicons/es/regular/car3";
import MultiIcon from "../MultiIcon";

import airpurifier from "./Airpurifier.vue";
import battery from "./Battery.vue";
import bike from "./Bike.vue";
import bulb from "./Bulb.vue";
import bus from "./Bus.vue";
import climate from "./Climate.vue";
import coffeemaker from "./Coffeemaker.vue";
import compute from "./Compute.vue";
import cooking from "./Cooking.vue";
import cooler from "./Cooler.vue";
import desktop from "./Desktop.vue";
import device from "./Device.vue";
import dishwasher from "./Dishwasher.vue";
import dryer from "./Dryer.vue";
import floorlamp from "./Floorlamp.vue";
import generic from "./Generic.vue";
import heater from "./Heater.vue";
import heatexchange from "./Heatexchange.vue";
import heatpump from "./Heatpump.vue";
import kettle from "./Kettle.vue";
import laundry from "./Laundry.vue";
import laundry2 from "./Laundry2.vue";
import machine from "./Machine.vue";
import meter from "./Meter.vue";
import microwave from "./Microwave.vue";
import moped from "./Moped.vue";
import motorcycle from "./Motorcycle.vue";
import pump from "./Pump.vue";
import rickshaw from "./Rickshaw.vue";
import rocket from "./Rocket.vue";
import scooter from "./Scooter.vue";
import shuttle from "./Shuttle.vue";
import taxi from "./Taxi.vue";
import tool from "./Tool.vue";
import tractor from "./Tractor.vue";
import van from "./Van.vue";
import waterheater from "./WaterHeater.vue";

const icons = {
	airpurifier,
	battery,
	bike,
	bulb,
	bus,
	car: "shopicon-regular-car3",
	climate,
	coffeemaker,
	compute,
	cooking,
	cooler,
	desktop,
	device,
	dishwasher,
	dryer,
	floorlamp,
	generic,
	heater,
	heatexchange,
	heatpump,
	kettle,
	laundry,
	laundry2,
	machine,
	meter,
	microwave,
	moped,
	motorcycle,
	pump,
	rickshaw,
	rocket,
	scooter,
	shuttle,
	taxi,
	tool,
	tractor,
	van,
	waterheater,
};

export default {
	name: "VehicleIcon",
	components: { MultiIcon },
	props: {
		name: { type: String, default: "car" },
		names: { type: Array },
		size: { type: String, default: "s" },
	},
	computed: {
		uniqueNames: function () {
			return [...new Set(this.names || [this.name])];
		},
		count: function () {
			return this.names?.length || 0;
		},
		single: function () {
			return this.uniqueNames.length == 1;
		},
		singleIcon: function () {
			return icons[this.uniqueNames[0]] || `shopicon-regular-car3`;
		},
	},
};
</script>

<style scoped>
.icon {
	display: block;
	width: 24px;
	height: 24px;
}
.icon--m {
	width: 32px;
	height: 32px;
}
.icon--l {
	width: 48px;
	height: 48px;
}
.icon--xl {
	width: 64px;
	height: 64px;
}
</style>
