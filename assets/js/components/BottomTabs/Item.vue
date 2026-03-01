<template>
	<component
		:is="to ? 'router-link' : 'div'"
		v-bind="linkProps"
		class="tab-item d-flex flex-column flex-md-row align-items-center justify-content-center gap-md-1 text-decoration-none position-relative"
		:class="{ active: !to && active }"
		:data-testid="label ? `tab-${label.toLowerCase()}` : undefined"
		@click="vibrate"
	>
		<slot />
		<span
			v-if="label"
			class="tab-label fw-bold mt-1 mt-md-0 text-truncate text-center text-md-start"
			>{{ label }}</span
		>
	</component>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { hapticFeedback } from "@/utils/haptic";

export default defineComponent({
	name: "BottomTabItem",
	props: {
		to: { type: String, default: undefined },
		label: { type: String, default: undefined },
		exact: { type: Boolean, default: false },
		active: { type: Boolean, default: false },
	},
	computed: {
		linkProps() {
			if (!this.to) return {};
			return {
				to: this.to,
				exactActiveClass: this.exact ? "active" : undefined,
				activeClass: this.exact ? undefined : "active",
			};
		},
	},
	methods: {
		vibrate() {
			hapticFeedback();
		},
	},
});
</script>

<style scoped>
.tab-item {
	--pt: 0.4rem;
	--pb: max(0.4rem, var(--safe-area-inset-bottom));
	flex: 1 1 0;
	min-width: 0;
	height: calc(var(--tab-bar-height) + var(--pb));
	padding-top: var(--pt);
	padding-bottom: var(--pb);
	color: var(--evcc-gray);
	touch-action: manipulation;
	-webkit-tap-highlight-color: transparent;
	-webkit-touch-callout: none;
	user-select: none;
	transition: color var(--evcc-transition-fast);
}

.tab-item::before {
	content: "";
	position: absolute;
	top: 0;
	left: 15%;
	width: 70%;
	height: 2px;
	border-radius: 0 0 2px 2px;
	background: transparent;
	transition: background var(--evcc-transition-fast);
}

.tab-item:hover {
	color: color-mix(in srgb, var(--evcc-gray) 70%, white);
}

.tab-item:active {
	color: color-mix(in srgb, var(--evcc-gray) 70%, black);
}

.tab-item.active {
	color: var(--bs-primary);
}

.tab-item.active::before {
	background: var(--bs-primary);
}

.tab-label {
	display: none;
	font-size: 10px;
	line-height: 1.2;
}

@media (min-width: 400px) {
	.tab-label {
		display: block;
	}
}
</style>
