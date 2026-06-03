<template>
	<span
		ref="root"
		class="d-flex align-items-center gap-2 text-nowrap evcc-gray"
		:role="tooltip ? 'img' : undefined"
		:aria-label="tooltip || undefined"
		:title="tooltip || undefined"
		:data-bs-toggle="tooltip ? 'tooltip' : undefined"
	>
		<span class="d-inline-block rounded-circle status-dot" :class="dotClass"></span>
		<slot />
	</span>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Tooltip from "bootstrap/js/dist/tooltip";

type Variant = "success" | "warning" | "muted";

export default defineComponent({
	name: "StatusIndicator",
	props: {
		variant: { type: String as PropType<Variant>, default: "muted" },
		tooltip: { type: String, default: "" },
	},
	data() {
		return { tooltipInstance: null as Tooltip | null };
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
	watch: {
		tooltip() {
			this.initTooltip();
		},
	},
	mounted() {
		this.initTooltip();
	},
	beforeUnmount() {
		this.tooltipInstance?.dispose();
	},
	methods: {
		initTooltip() {
			this.$nextTick(() => {
				this.tooltipInstance?.dispose();
				this.tooltipInstance = null;
				const el = this.$refs["root"] as Element | undefined;
				if (el && this.tooltip) {
					// explicit title: bootstrap clears the attr, so :title alone goes stale
					this.tooltipInstance = new Tooltip(el, { title: this.tooltip });
				}
			});
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
