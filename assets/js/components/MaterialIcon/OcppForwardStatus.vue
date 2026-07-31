<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<!-- configured and connected: cloud with check -->
		<path
			v-if="status === 'configured'"
			fill="currentColor"
			d="m10.325 14.125l-1.4-1.4q-.3-.3-.7-.3t-.7.3t-.3.713t.3.712L9.65 16.3q.3.3.7.3t.7-.3l4.225-4.225q.3-.3.3-.725t-.3-.725t-.725-.3t-.725.3zM6.5 20q-2.275 0-3.887-1.575T1 14.575q0-1.95 1.175-3.475T5.25 9.15q.625-2.3 2.5-3.725T12 4q2.925 0 4.963 2.038T19 11q1.725.2 2.863 1.488T23 15.5q0 1.875-1.312 3.188T18.5 20zm0-2h12q1.05 0 1.775-.725T21 15.5t-.725-1.775T18.5 13H17v-2q0-2.075-1.463-3.538T12 6T8.463 7.463T7 11h-.5q-1.45 0-2.475 1.025T3 14.5t1.025 2.475T6.5 18m5.5-6"
		/>
		<!-- configured but not connected: cloud with alert -->
		<path
			v-else-if="status === 'error'"
			fill="currentColor"
			d="M6.5 20q-2.275 0-3.887-1.575T1 14.575q0-1.95 1.175-3.475T5.25 9.15q.625-2.3 2.5-3.725T12 4q2.925 0 4.963 2.038T19 11q1.725.2 2.863 1.488T23 15.5q0 1.875-1.312 3.188T18.5 20zm0-2h12q1.05 0 1.775-.725T21 15.5t-.725-1.775T18.5 13H17v-2q0-2.075-1.463-3.538T12 6T8.463 7.463T7 11h-.5q-1.45 0-2.475 1.025T3 14.5t1.025 2.475T6.5 18m5.5-2q.425 0 .713-.288T13 15t-.288-.712T12 14t-.712.288T11 15t.288.713T12 16m0-3.5q.425 0 .713-.288T13 11.5V9q0-.425-.288-.712T12 8t-.712.288T11 9v2.5q0 .425.288.713T12 12.5"
		/>
		<!-- no forwarding configured: plain cloud -->
		<path
			v-else
			fill="currentColor"
			d="M6.5 20q-2.275 0-3.887-1.575T1 14.575q0-1.95 1.175-3.475T5.25 9.15q.625-2.3 2.5-3.725T12 4q2.925 0 4.963 2.038T19 11q1.725.2 2.863 1.488T23 15.5q0 1.875-1.312 3.188T18.5 20zm0-2h12q1.05 0 1.775-.725T21 15.5t-.725-1.775T18.5 13H17v-2q0-2.075-1.463-3.538T12 6T8.463 7.463T7 11h-.5q-1.45 0-2.475 1.025T3 14.5t1.025 2.475T6.5 18m5.5-6"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import icon from "@/mixins/icon";

type ForwardStatus = "unconfigured" | "configured" | "error";

export default defineComponent({
	name: "OcppForwardStatus",
	mixins: [icon],
	props: {
		status: { type: String as PropType<ForwardStatus>, default: "unconfigured" },
	},
});
</script>
