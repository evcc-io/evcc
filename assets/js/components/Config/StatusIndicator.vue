<template>
	<span class="d-flex align-items-center gap-2 text-nowrap evcc-gray">
		<span class="d-inline-block p-1 rounded-circle" :class="dotClass"></span>
		<slot />
	</span>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";

type Variant = "success" | "warning" | "muted";

export default defineComponent({
	name: "StatusIndicator",
	props: {
		variant: { type: String as PropType<Variant>, default: "muted" },
	},
	computed: {
		dotClass(): string {
			switch (this.variant) {
				case "success":
					return "bg-success";
				case "warning":
					return "bg-warning";
				default:
					return "border border-secondary";
			}
		},
	},
});
</script>
