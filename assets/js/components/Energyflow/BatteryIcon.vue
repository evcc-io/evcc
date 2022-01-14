<template>
	<div :class="{ 'power--in': charge, 'power--out': discharge }" class="d-flex">
		<shopicon-regular-batteryfull class="battery flex-shrink-0"></shopicon-regular-batteryfull
		><shopicon-regular-angledoublerightsmall
			class="arrow flex-shrink-0"
		></shopicon-regular-angledoublerightsmall>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/angledoublerightsmall";
import "@h2d2/shopicons/es/regular/batteryfull";
import "@h2d2/shopicons/es/regular/batterythreequarters";
import "@h2d2/shopicons/es/regular/batteryhalf";
import "@h2d2/shopicons/es/regular/batteryquarter";
import "@h2d2/shopicons/es/regular/batteryempty";

export default {
	name: "BatteryIcon",
	props: {
		discharge: { type: Boolean },
		charge: { type: Boolean },
		soc: { type: Number, default: 0 },
	},
	computed: {
		batteryIcon: function () {
			if (this.soc > 80) return "battery-full";
			if (this.soc > 60) return "battery-three-quarters";
			if (this.soc > 40) return "battery-half";
			if (this.soc > 20) return "battery-quarter";
			return "battery-empty";
		},
	},
};
</script>
<style scoped>
.battery {
	transform: translateX(0.35rem);
	transition-property: transform;
	transition-duration: 250ms;
	transition-timing-function: ease;
}
.power--in .battery {
	transform: translateX(0.7rem);
}
.power--out .battery {
	transform: translateX(0);
}
.arrow {
	margin-left: -0.2rem;
	opacity: 0;
	transform: translateX(-0.5rem);
	transition-property: opacity, transform;
	transition-duration: 250ms;
	transition-timing-function: ease;
}
.power--in .arrow {
	opacity: 1;
	transform: translateX(-1rem) scaleX(1);
}
.power--out .arrow {
	opacity: 1;
	transform: translateX(0rem) scaleX(1);
}
</style>
