<template>
	<div class="mb-2 entry" :class="{ 'evcc-gray': !active }">
		<div class="d-flex justify-content-between">
			<span class="d-flex flex-nowrap">
				<BatteryIcon v-if="isBattery" v-bind="iconProps" />
				<VehicleIcon v-else-if="isVehicle" v-bind="iconProps" />
				<component :is="`shopicon-regular-${icon}`" v-else></component>
			</span>
			<div class="d-block text-nowrap flex-grow-1 ms-3 text-truncate">
				{{ name }}
			</div>
			<span class="text-end text-nowrap ps-1 fw-bold d-flex">
				<div
					ref="details"
					class="fw-normal"
					:class="{ 'text-decoration-underline': detailsClickable }"
					data-testid="energyflow-entry-details"
					data-bs-toggle="tooltip"
					:tabindex="detailsClickable ? 0 : undefined"
					@click="detailsClicked"
				>
					<AnimatedNumber v-if="!isNaN(details)" :to="details" :format="detailsFmt" />
				</div>
				<div ref="power" class="power" data-bs-toggle="tooltip" @click="powerClicked">
					<AnimatedNumber ref="powerNumber" :to="power" :format="kw" />
				</div>
			</span>
		</div>
		<div v-if="$slots.subline" class="ms-4 ps-3">
			<slot name="subline" />
		</div>
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

export default {
	name: "EnergyflowEntry",
	components: { BatteryIcon, AnimatedNumber, VehicleIcon },
	mixins: [formatter],
	props: {
		name: { type: String },
		icon: { type: String },
		iconProps: { type: Object, default: () => ({}) },
		power: { type: Number },
		powerTooltip: { type: Array },
		powerUnit: { type: String },
		details: { type: Number },
		detailsFmt: { type: Function },
		detailsTooltip: { type: Array },
		detailsClickable: { type: Boolean },
	},
	emits: ["details-clicked"],
	data() {
		return { powerTooltipInstance: null, detailsTooltipInstance: null };
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
	},
	watch: {
		powerTooltip(newVal, oldVal) {
			if (JSON.stringify(newVal) !== JSON.stringify(oldVal)) {
				this.updatePowerTooltip();
			}
		},
		detailsTooltip(newVal, oldVal) {
			if (JSON.stringify(newVal) !== JSON.stringify(oldVal)) {
				this.updateDetailsTooltip();
			}
		},
		powerInKw(newVal, oldVal) {
			// force update if unit changes but not the value
			if (newVal !== oldVal) {
				this.$refs.powerNumber.forceUpdate();
			}
		},
	},
	mounted: function () {
		this.updatePowerTooltip();
		this.updateDetailsTooltip();
	},
	methods: {
		kw: function (watt) {
			return this.fmtW(watt, this.powerUnit);
		},
		updatePowerTooltip() {
			this.powerTooltipInstance = this.updateTooltip(
				this.powerTooltipInstance,
				this.powerTooltip,
				this.$refs.power
			);
		},
		updateDetailsTooltip() {
			if (this.detailsClickable) {
				return;
			}
			this.detailsTooltipInstance = this.updateTooltip(
				this.detailsTooltipInstance,
				this.detailsTooltip,
				this.$refs.details
			);
		},
		updateTooltip: function (instance, content, ref) {
			if (!Array.isArray(content) || !content.length) {
				if (instance) {
					instance.dispose();
				}
				return;
			}
			let newInstance = instance;
			if (!newInstance) {
				newInstance = new Tooltip(ref, { html: true, title: " " });
			}
			const html = `<div class="text-end">${content.join("<br/>")}</div>`;
			newInstance.setContent({ ".tooltip-inner": html });
			return newInstance;
		},
		powerClicked: function ($event) {
			if (this.powerTooltip) {
				$event.stopPropagation();
			}
		},
		detailsClicked: function ($event) {
			if (this.detailsClickable || this.detailsTooltip) {
				$event.stopPropagation();
			}
			if (this.detailsClickable) {
				this.$emit("details-clicked");
			}
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
