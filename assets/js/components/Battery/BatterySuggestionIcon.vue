<template>
	<!-- normal: free charge/discharge -->
	<BatteryNormal v-if="action === 'normal'" :size="size" />
	<!-- charge: grid charge -->
	<shopicon-regular-powersupply
		v-else-if="action === 'charge'"
		:size="size"
	></shopicon-regular-powersupply>
	<!-- holdcharge: prevent charging -->
	<BatteryHoldCharge v-else-if="action === 'holdcharge'" :size="size" />
	<!-- discharge: battery-to-grid export -->
	<shopicon-regular-powersupply
		v-else-if="action === 'discharge'"
		:size="size"
	></shopicon-regular-powersupply>
	<!-- hold: prevent discharging -->
	<BatteryHold v-else :size="size" />
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import "@h2d2/shopicons/es/regular/powersupply";
import { ICON_SIZE } from "@/types/evcc";
import type { BatterySuggestion } from "@/types/evcc";
import BatteryNormal from "../MaterialIcon/BatteryNormal.vue";
import BatteryHold from "../MaterialIcon/BatteryHold.vue";
import BatteryHoldCharge from "../MaterialIcon/BatteryHoldCharge.vue";

export default defineComponent({
	name: "BatterySuggestionIcon",
	components: { BatteryNormal, BatteryHold, BatteryHoldCharge },
	props: {
		action: { type: String as PropType<BatterySuggestion["action"]>, required: true },
		size: { type: String as PropType<ICON_SIZE>, default: ICON_SIZE.S },
	},
});
</script>
