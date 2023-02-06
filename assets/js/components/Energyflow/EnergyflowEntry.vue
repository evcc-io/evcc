<template>
	<div class="d-flex justify-content-between mb-2 entry" :class="{ 'evcc-gray': !active }">
		<span class="d-flex flex-nowrap">
			<BatteryIcon v-if="isBattery" :soc="soc" />
			<VehicleIcon v-else-if="isVehicle" :names="vehicleIcons" />
			<component :is="`shopicon-regular-${icon}`" v-else></component>
		</span>
		<span class="text-nowrap flex-grow-1 ms-3">
			{{ name }}
		</span>
		<span class="text-end text-nowrap ps-1 fw-bold d-flex">
			<div
				ref="details"
				class="evcc-gray fw-normal"
				data-bs-toggle="tooltip"
				title=" "
				@click.stop=""
			>
				<span v-if="showPrice()"><AnimatedNumber :to="price" :format="fmtPrice" /></span>
				<span v-if="showCo2()"><AnimatedNumber :to="co2" :format="fmtCo2Short" /></span>
				<span v-if="hasSoc">{{ soc }}%</span>
			</div>
			<AnimatedNumber :to="power" :format="kw" class="power" />
		</span>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/home";
import Tooltip from "bootstrap/js/dist/tooltip";
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
		co2: { type: Number },
		valuesInKw: { type: Boolean },
		vehicleIcons: { type: Array },
		currency: { type: String },
		tooltip: { type: Array },
	},
	data() {
		return { tooltipInstance: null };
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
	watch: {
		tooltip(newVal, oldVal) {
			if (JSON.stringify(newVal) !== JSON.stringify(oldVal)) {
				this.updateTooltip();
			}
		},
	},
	mounted: function () {
		this.updateTooltip();
	},
	methods: {
		showPrice() {
			return showGridPrice() && !isNaN(this.price);
		},
		showCo2() {
			return showGridCo2() && !isNaN(this.co2);
		},
		kw: function (watt) {
			return this.fmtKw(watt, this.valuesInKw);
		},
		fmtPrice: function (price) {
			return this.fmtPricePerKWh(price, this.currency, true);
		},
		updateTooltip: function () {
			if (!Array.isArray(this.tooltip) || !this.tooltip.length) {
				return;
			}
			if (!this.tooltipInstance) {
				this.tooltipInstance = new Tooltip(this.$refs.details, { html: true });
			}
			const html = `<div class="text-end">${this.tooltip.join("<br/>")}</div>`;
			this.tooltipInstance.setContent({ ".tooltip-inner": html });
		},
	},
};
</script>
<style scoped>
.entry {
	transition: color var(--evcc-transition-medium) linear;
}
.power {
	min-width: 75px;
}
</style>
