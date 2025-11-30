<template>
	<div>
		<div class="mb-2 entry" :class="{ 'evcc-gray': !active }">
			<div class="d-flex justify-content-between">
				<span class="d-flex flex-nowrap">
					<BatteryIcon v-if="isBattery" v-bind="iconProps" />
					<VehicleIcon v-else-if="isVehicle" v-bind="iconProps" />
					<div v-else-if="!icon" class="icon-placeholder"></div>
					<component :is="`shopicon-regular-${icon}`" v-else></component>
				</span>
				<div class="d-flex flex-grow-1 ms-3 align-items-center text-truncate">
					<span v-if="!$slots['expanded']" class="text-truncate">
						{{ name }}
					</span>
					<button
						v-else
						class="btn-neutral d-flex align-items-center flex-shrink-1 flex-grow-1"
						style="max-width: 100%"
						@click="toggle"
					>
						<div class="flex-shrink-1 flex-grow-0 d-flex text-truncate">
							<span class="text-truncate"> {{ name }} </span>
						</div>
						<shopicon-regular-arrowdropdown
							class="expand-icon flex-shrink-0 flex-grow-0"
							:class="{ 'expand-icon--expanded': expanded }"
						/>
					</button>
				</div>
				<span class="text-end text-nowrap ps-1 fw-bold d-flex align-items-center">
					<div
						ref="details"
						class="fw-normal d-flex align-items-center"
						:class="{
							'text-decoration-underline': detailsClickable,
							'evcc-gray': detailsInactive,
						}"
						data-testid="energyflow-entry-details"
						data-bs-toggle="tooltip"
						:tabindex="detailsClickable ? 0 : undefined"
						@click="detailsClicked"
					>
						<ForecastIcon
							v-if="detailsIcon === 'forecast'"
							class="ms-2 me-1 d-inline-block"
						/>
						<AnimatedNumber
							v-if="details !== undefined && !isNaN(details)"
							:to="details"
							:format="detailsFmt!"
						/>
					</div>
					<div ref="power" class="power" data-bs-toggle="tooltip" @click="powerClicked">
						<AnimatedNumber ref="powerNumber" :to="power" :format="kw" />
					</div>
				</span>
			</div>
		</div>
		<div
			v-if="$slots['expanded']"
			class="expandable ms-2"
			:class="{ 'expandable--open': expanded }"
		>
			<slot name="expanded" />
		</div>
		<div v-if="$slots['subline']" class="ms-4 ps-3 mb-2">
			<slot name="subline" />
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/home";
import "@h2d2/shopicons/es/regular/arrowdropdown";
import Tooltip from "bootstrap/js/dist/tooltip";
import BatteryIcon from "./BatteryIcon.vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import AnimatedNumber from "../Helper/AnimatedNumber.vue";
import VehicleIcon from "../VehicleIcon";
import ForecastIcon from "../MaterialIcon/Forecast.vue";
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	name: "EnergyflowEntry",
	components: { BatteryIcon, AnimatedNumber, VehicleIcon, ForecastIcon },
	mixins: [formatter],
	props: {
		name: { type: String },
		icon: { type: String },
		iconProps: { type: Object, default: () => ({}) },
		power: { type: Number, default: 0 },
		powerTooltip: { type: Array as PropType<string[]> },
		powerUnit: { type: String as PropType<POWER_UNIT> },
		details: { type: Number },
		detailsIcon: { type: String },
		detailsFmt: { type: Function as PropType<(n: number) => string> },
		detailsTooltip: { type: Array as PropType<string[]> },
		detailsClickable: { type: Boolean },
		detailsInactive: { type: Boolean },
		expanded: { type: Boolean, default: false },
	},
	emits: ["details-clicked", "toggle"],
	data() {
		return {
			powerTooltipInstance: null as Tooltip | null,
			detailsTooltipInstance: null as Tooltip | null,
		};
	},
	computed: {
		active() {
			return this.power > 10;
		},
		isBattery() {
			return this.icon === "battery";
		},
		isVehicle() {
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
		powerUnit(newVal, oldVal) {
			// force update if unit changes but not the value
			if (newVal !== oldVal) {
				(this.$refs["powerNumber"] as any).forceUpdate();
			}
		},
	},
	mounted() {
		this.updatePowerTooltip();
		this.updateDetailsTooltip();
	},
	methods: {
		kw(watt: number) {
			return this.fmtW(watt, this.powerUnit);
		},
		updatePowerTooltip() {
			this.powerTooltipInstance = this.updateTooltip(
				this.powerTooltipInstance,
				this.$refs["power"],
				this.powerTooltip
			);
		},
		updateDetailsTooltip() {
			this.detailsTooltipInstance = this.updateTooltip(
				this.detailsTooltipInstance,
				this.$refs["details"],
				this.detailsTooltip
			);
		},
		updateTooltip(instance: Tooltip | null, ref: any, content?: string[]) {
			if (!Array.isArray(content) || !content.length) {
				if (instance) {
					instance.dispose();
				}
				return null;
			}
			let newInstance = instance;
			if (!newInstance) {
				newInstance = new Tooltip(ref, { html: true, title: " " });
			}
			const html = `<div class="text-end">${content.join("<br/>")}</div>`;
			newInstance.setContent({ ".tooltip-inner": html });
			return newInstance;
		},
		powerClicked($event: Event) {
			if (this.powerTooltip) {
				$event.stopPropagation();
			}
		},
		detailsClicked($event: Event) {
			if (this.detailsClickable || this.detailsTooltip) {
				$event.stopPropagation();
			}
			if (this.detailsClickable) {
				this.$emit("details-clicked");
			}
			// hide tooltip, chrome needs a timeout
			setTimeout(() => this.detailsTooltipInstance?.hide(), 10);
		},
		toggle($event: Event) {
			$event.stopPropagation();
			this.$emit("toggle");
		},
	},
});
</script>
<style scoped>
.entry {
	transition: color var(--evcc-transition-medium) linear;
}
.power {
	min-width: 75px;
}
.icon-placeholder {
	width: 24px;
	aspect-ratio: 1;
}
.expand-icon {
	transition: transform var(--evcc-transition-medium) ease;
	transform: rotate(-90deg);
}
.expand-icon--expanded {
	transform: rotate(0deg);
}
.expandable {
	overflow: hidden;
	opacity: 0;
	height: 0;
	transition: opacity var(--evcc-transition-medium) ease-in;
}
.expandable--open {
	opacity: 1;
	height: auto;
}
</style>
