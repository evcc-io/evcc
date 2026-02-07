<template>
	<DeviceCard
		:title="cardTitle"
		:name="meter.name"
		:editable="!!meter.id"
		:error="hasError"
		:data-testid="meterType"
		@edit="$emit('edit', meterType, meter.id)"
	>
		<template #icon>
			<VehicleIcon v-if="isVehicleIcon" :name="iconName" />
			<component :is="iconComponent" v-else />
		</template>
		<template #tags>
			<DeviceTags :tags="tags" />
		</template>
	</DeviceCard>
</template>

<script>
import DeviceCard from "./DeviceCard.vue";
import DeviceTags from "./DeviceTags.vue";
import VehicleIcon from "../VehicleIcon";

export default {
	name: "MeterCard",
	components: {
		DeviceCard,
		DeviceTags,
		VehicleIcon,
	},
	props: {
		meter: {
			type: Object,
			required: true,
		},
		meterType: {
			type: String,
			required: true,
			validator: (value) => ["grid", "pv", "battery", "aux", "ext"].includes(value),
		},
		hasError: {
			type: Boolean,
			default: false,
		},
		title: {
			type: String,
		},
		tags: {
			type: Object,
			default: () => ({}),
		},
	},
	emits: ["edit"],
	computed: {
		cardTitle() {
			if (this.title) {
				return this.title;
			}
			if (this.meter.deviceTitle) {
				return this.meter.deviceTitle;
			}
			if (this.meter.config?.template) {
				return this.meter.config.template;
			}
			return this.fallbackTitle;
		},
		fallbackTitle() {
			const titleMap = {
				grid: this.$t("config.grid.title"),
				pv: this.$t("config.devices.solarSystem"),
				battery: this.$t("config.devices.batteryStorage"),
				aux: this.$t("config.devices.auxMeter"),
				ext: this.$t("config.devices.extMeter"),
			};
			return titleMap[this.meterType];
		},
		isVehicleIcon() {
			return this.meterType === "aux" || this.meterType === "ext";
		},
		iconComponent() {
			const iconMap = {
				grid: "shopicon-regular-powersupply",
				pv: "shopicon-regular-sun",
				battery: "shopicon-regular-batterythreequarters",
			};
			return iconMap[this.meterType];
		},
		iconName() {
			if (this.meterType === "aux") {
				return this.meter.deviceIcon || "smartconsumer";
			}
			if (this.meterType === "ext") {
				return this.meter.deviceIcon || "generic";
			}
			return null;
		},
	},
};
</script>
