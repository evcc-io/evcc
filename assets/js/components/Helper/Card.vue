<template>
	<section class="evcc-card p-3 p-sm-4" :class="{ 'evcc-card--edge': edgeToEdge }">
		<div
			v-if="hasHeader"
			class="evcc-card-header d-flex align-items-center gap-3 flex-wrap mb-3"
		>
			<h3
				v-if="hasTitle || hasSubtitle"
				class="evcc-card-title fw-normal m-0 d-flex gap-3 flex-wrap align-items-baseline overflow-hidden"
			>
				<span class="d-block no-wrap text-truncate">
					<slot name="title">{{ title }}</slot>
				</span>
				<small v-if="hasSubtitle" class="d-block no-wrap text-truncate evcc-card-subtitle">
					<slot name="subtitle">{{ subtitle }}</slot>
				</small>
			</h3>
			<slot name="nav" />
			<div v-if="$slots['actions']" class="ms-auto">
				<slot name="actions" />
			</div>
		</div>
		<slot />
	</section>
</template>

<script lang="ts">
import { defineComponent } from "vue";

// Reusable content card: optional header (title + subtitle + nav/actions slots) over a
// rounded box. `edgeToEdge` makes it full-bleed on mobile (for charts that need width).
// Mirrors the History tile style so the look stays consistent across pages.
export default defineComponent({
	name: "Card",
	props: {
		title: { type: String, default: "" },
		subtitle: { type: String, default: "" },
		edgeToEdge: Boolean,
	},
	computed: {
		hasTitle(): boolean {
			return !!this.title || !!this.$slots["title"];
		},
		hasSubtitle(): boolean {
			return !!this.subtitle || !!this.$slots["subtitle"];
		},
		hasHeader(): boolean {
			return (
				this.hasTitle ||
				this.hasSubtitle ||
				!!this.$slots["nav"] ||
				!!this.$slots["actions"]
			);
		},
	},
});
</script>

<style scoped>
.evcc-card {
	background: var(--evcc-box);
	border: 1px solid var(--bs-border-color-translucent);
	border-radius: 1rem;
}
.evcc-card-subtitle {
	color: var(--evcc-gray);
}
@media (max-width: 575.98px) {
	.evcc-card--edge {
		margin-left: -1.5rem;
		margin-right: -1.5rem;
		border-left: none;
		border-right: none;
		border-radius: 0;
	}
}
</style>
