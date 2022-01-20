<template>
	<component :is="`shopicon-regular-${icon}`"></component>
</template>

<script>
import "@h2d2/shopicons/es/regular/batteryfull";
import "@h2d2/shopicons/es/regular/batterythreequarters";
import "@h2d2/shopicons/es/regular/batteryhalf";
import "@h2d2/shopicons/es/regular/batteryquarter";
import "@h2d2/shopicons/es/regular/batteryempty";

export default {
	name: "BatteryIcon",
	props: {
		soc: { type: Number, default: 0 },
	},
	computed: {
		icon: function () {
			if (this.soc > 80) return "batteryfull";
			if (this.soc > 60) return "batterythreequarters";
			if (this.soc > 40) return "batteryhalf";
			if (this.soc > 20) return "batteryquarter";
			return "batteryempty";
		},
	},
};
</script>
