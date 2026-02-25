<template>
	<router-link
		:to="to"
		class="tab-item d-flex flex-column flex-md-row align-items-center justify-content-center gap-md-1 text-decoration-none position-relative"
		:exact-active-class="exactActiveClass"
		:active-class="activeClass"
	>
		<slot />
		<span
			class="tab-label fw-bold text-uppercase mt-1 mt-md-0 text-truncate text-center text-md-start"
			>{{ label }}</span
		>
	</router-link>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "BottomTabItem",
	props: {
		to: { type: String, required: true },
		label: { type: String, required: true },
		exact: { type: Boolean, default: false },
	},
	computed: {
		exactActiveClass(): string | undefined {
			return this.exact ? "active" : undefined;
		},
		activeClass(): string | undefined {
			return this.exact ? undefined : "active";
		},
	},
});
</script>

<style scoped>
.tab-item {
	flex: 1 1 0;
	min-width: 0;
	padding: 6px 0;
	color: var(--evcc-gray);
	border-top: 2px solid transparent;
	touch-action: manipulation;
	-webkit-tap-highlight-color: transparent;
	transition:
		color var(--evcc-transition-very-fast),
		border-color var(--evcc-transition-very-fast);
}

.tab-item:hover {
	color: color-mix(in srgb, var(--evcc-gray) 70%, white);
}

.tab-item:active {
	color: color-mix(in srgb, var(--evcc-gray) 70%, black);
}

.tab-item.active {
	color: var(--bs-primary);
	border-top-color: var(--bs-primary);
}

.tab-label {
	font-size: 10px;
	line-height: 1.2;
}

:deep(.tab-icon) {
	width: 24px;
	height: 24px;
	display: block;
}

@media (--md-and-up) {
	:deep(.tab-icon) {
		width: 18px;
		height: 18px;
	}
	.tab-label {
		font-size: 11px;
	}
}
</style>
