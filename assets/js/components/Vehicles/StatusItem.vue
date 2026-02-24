<template>
	<button v-if="clickable" type="button" class="entry" @click="handleClick">
		<component :is="iconComponent" v-if="iconComponent" :class="iconClass" />
		<div v-if="hasContent" :class="{ tabular }">
			<slot>
				{{ content }}
			</slot>
		</div>
	</button>
	<div v-else class="entry">
		<component :is="iconComponent" v-if="iconComponent" :class="iconClass" />
		<div v-if="hasContent" :class="{ tabular }">
			<slot>
				{{ content }}
			</slot>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import Tooltip from "bootstrap/js/dist/tooltip";

export default defineComponent({
	name: "StatusItem",
	props: {
		content: { type: String, default: "" },
		tooltipContent: { type: String, default: "" },
		iconComponent: { type: [String, Object], default: null },
		iconClass: { type: String, default: "" },
		clickable: { type: Boolean, default: false },
		tabular: { type: Boolean, default: false },
	},
	emits: ["click"],
	data() {
		return {
			tooltip: null as Tooltip | null,
		};
	},
	computed: {
		hasContent() {
			return this.content || this.$slots["default"];
		},
	},
	watch: {
		tooltipContent() {
			this.$nextTick(this.updateTooltip);
		},
	},
	mounted() {
		this.updateTooltip();
	},
	beforeUnmount() {
		if (this.tooltip) {
			this.tooltip.dispose();
		}
	},
	methods: {
		handleClick() {
			this.tooltip?.hide();
			this.$emit("click");
		},
		updateTooltip() {
			if (!this.tooltipContent || !this.$el) {
				if (this.tooltip) {
					this.tooltip.dispose();
					this.tooltip = null;
				}
				return;
			}

			if (!this.tooltip) {
				this.tooltip = new Tooltip(this.$el, {
					title: " ",
					trigger: "hover",
				});
			}

			this.tooltip.setContent({ ".tooltip-inner": this.tooltipContent });
		},
	},
});
</script>

<style scoped>
.entry {
	display: flex;
	align-items: center;
	flex-wrap: nowrap;
	text-wrap: nowrap;
	border: none;
	color: inherit;
	background: none;
	padding: 0;
	gap: 0.5rem;
	transition:
		color var(--evcc-transition-medium) linear,
		opacity var(--evcc-transition-medium) linear;
}

.phaseUp {
	transform: rotate(90deg);
}

.phaseDown {
	transform: rotate(-90deg);
}
.tabular {
	font-variant-numeric: tabular-nums;
}
</style>
