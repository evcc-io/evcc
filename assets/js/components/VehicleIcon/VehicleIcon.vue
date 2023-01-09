<template>
	<component :is="singleIcon" v-if="single" :class="`icon icon--${size}`"></component>
	<MultiIcon v-else :count="count" :size="size"></MultiIcon>
</template>

<script>
import "@h2d2/shopicons/es/regular/car3";
import MultiIcon from "../MultiIcon";

import bike from "./Bike.vue";
import bus from "./Bus.vue";
import cooler from "./Cooler.vue";
import generic from "./Generic.vue";
import heater from "./Heater.vue";
import moped from "./Moped.vue";
import motorcycle from "./Motorcycle.vue";
import scooter from "./Scooter.vue";
import tractor from "./Tractor.vue";
import van from "./Van.vue";
import waterheater from "./WaterHeater.vue";

const icons = {
	bike,
	bus,
	car: "shopicon-regular-car3",
	cooler,
	generic,
	heater,
	moped,
	motorcycle,
	scooter,
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
			return this.names.length;
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
