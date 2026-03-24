<template>
	<button
		type="button"
		class="btn btn-sm btn-outline-secondary border-0 p-2 evcc-gray flex-shrink-0 loadpoint-expand-btn"
		:aria-label="
			expanded ? $t('main.loadpoint.backToOverview') : $t('main.loadpoint.expandViewport')
		"
		:data-testid="
			expanded
				? mobile
					? 'loadpoint-viewport-back-xs'
					: 'loadpoint-viewport-back'
				: mobile
					? 'loadpoint-viewport-expand-xs'
					: 'loadpoint-viewport-expand'
		"
		@click.stop="$emit('toggle')"
	>
		<svg
			v-if="expanded"
			class="loadpoint-expand-icon"
			width="18"
			height="18"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
			aria-hidden="true"
			focusable="false"
		>
			<path d="M4 14h6v6" />
			<path d="M20 10h-6V4" />
			<path d="M14 10l7-7" />
			<path d="M3 21l7-7" />
		</svg>
		<svg
			v-else
			class="loadpoint-expand-icon"
			width="18"
			height="18"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
			aria-hidden="true"
			focusable="false"
		>
			<path d="M8 3H5a2 2 0 0 0-2 2v3" />
			<path d="M21 8V5a2 2 0 0 0-2-2h-3" />
			<path d="M3 16v3a2 2 0 0 0 2 2h3" />
			<path d="M16 21h3a2 2 0 0 0 2-2v-3" />
		</svg>
	</button>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "ViewportToggleButton",
	props: {
		expanded: Boolean,
		mobile: Boolean,
	},
	emits: ["toggle"],
});
</script>

<style scoped>
.loadpoint-expand-icon {
	display: block;
}
</style>
