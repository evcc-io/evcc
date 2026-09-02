<template>
	<span
		v-tooltip="tooltip"
		class="d-flex align-items-center gap-2 text-nowrap evcc-gray"
		:role="tooltip ? 'img' : undefined"
		:aria-label="tooltip || undefined"
	>
		<span class="d-inline-block rounded-circle status-dot" :class="dotClass"></span>
		<slot />
	</span>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";

type Variant = "success" | "warning" | "danger" | "muted";

export default defineComponent({
	name: "StatusIndicator",
	props: {
		variant: { type: String as PropType<Variant>, default: "muted" },
		tooltip: { type: String, default: "" },
	},
	computed: {
		dotClass(): string {
			switch (this.variant) {
				case "success":
					return "bg-success";
				case "warning":
					return "bg-warning";
				case "danger":
					return "bg-danger";
				default:
					return "border border-secondary";
			}
		},
	},
});
</script>

<style scoped>
.status-dot {
	width: 0.8rem;
	height: 0.8rem;
	box-sizing: border-box;
}
</style>
