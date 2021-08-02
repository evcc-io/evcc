<template>
	<div>
		<fa-icon class="battery" :icon="batteryIcon"></fa-icon
		><fa-icon
			class="arrow"
			icon="angle-double-left"
			:class="{ 'arrow--in': charge, 'arrow--out': discharge }"
		></fa-icon>
	</div>
</template>

<script>
import "../../icons";

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
	transform: rotate(-90deg);
}
.arrow {
	margin-left: -0.25rem;
	opacity: 0;
	transform: scaleX(0);
	transition-property: opacity, transform;
	transition-duration: 250ms;
	transition-timing-function: ease;
}
.arrow--in {
	opacity: 1;
	transform: scaleX(1);
}
.arrow--out {
	opacity: 1;
	transform: scaleX(-1);
}
</style>
