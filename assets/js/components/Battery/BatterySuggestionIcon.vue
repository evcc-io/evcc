<template>
	<!-- normal: free charge/discharge -->
	<BatteryNormal v-if="action === 'normal'" :size="size" />
	<!-- charge: grid charge -->
	<BatteryCharge v-else-if="action === 'charge'" :size="size" />
	<!-- holdcharge: prevent charging -->
	<BatteryHoldCharge v-else-if="action === 'holdcharge'" :size="size" />
	<!-- discharge: battery-to-grid export -->
	<BatteryDischarge v-else-if="action === 'discharge'" :size="size" />
	<!-- hold: prevent discharging -->
	<BatteryHold v-else :size="size" />
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { ICON_SIZE } from "@/types/evcc";
import type { BatterySuggestion } from "@/types/evcc";
import BatteryNormal from "../MaterialIcon/BatteryNormal.vue";
import BatteryHold from "../MaterialIcon/BatteryHold.vue";
import BatteryHoldCharge from "../MaterialIcon/BatteryHoldCharge.vue";
import BatteryCharge from "../MaterialIcon/BatteryCharge.vue";
import BatteryDischarge from "../MaterialIcon/BatteryDischarge.vue";

export default defineComponent({
	name: "BatterySuggestionIcon",
	components: { BatteryNormal, BatteryHold, BatteryHoldCharge, BatteryCharge, BatteryDischarge },
	props: {
		action: { type: String as PropType<BatterySuggestion["action"]>, required: true },
		size: { type: String as PropType<ICON_SIZE>, default: ICON_SIZE.S },
	},
});
</script>
