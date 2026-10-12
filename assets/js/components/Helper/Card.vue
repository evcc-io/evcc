<template>
	<section class="evcc-card round-box p-3 p-sm-4" :class="{ 'evcc-card--edge': edgeToEdge }">
		<CardHeader v-if="hasHeader" class="mb-3" :title="title" :subtitle="subtitle">
			<template v-if="$slots['icon']" #icon><slot name="icon" /></template>
			<template v-if="$slots['nav']" #nav><slot name="nav" /></template>
			<template v-if="$slots['actions']" #actions><slot name="actions" /></template>
		</CardHeader>
		<slot />
	</section>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import CardHeader from "./CardHeader.vue";

// Reusable content card: optional header over a rounded box. `edgeToEdge` makes it
// full-bleed on mobile (for charts that need width).
export default defineComponent({
	name: "Card",
	components: { CardHeader },
	props: {
		title: { type: String, default: "" },
		subtitle: { type: String, default: "" },
		edgeToEdge: Boolean,
	},
	computed: {
		hasHeader(): boolean {
			return (
				!!this.title || !!this.subtitle || !!this.$slots["nav"] || !!this.$slots["actions"]
			);
		},
	},
});
</script>

<style scoped>
@media (max-width: 575.98px) {
	.evcc-card--edge {
		margin-left: -1.5rem;
		margin-right: -1.5rem;
		border-width: 1px 0;
		border-radius: 0;
		padding-left: 1.5rem !important;
		padding-right: 1.5rem !important;
	}
}
</style>
