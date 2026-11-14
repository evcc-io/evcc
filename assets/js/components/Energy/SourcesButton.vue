<template>
	<button
		v-if="hasOwnSources"
		type="button"
		class="sources btn btn-link d-flex rounded-circle"
		:aria-label="$t('energy.sources.title')"
		@click="$emit('click')"
	>
		<ShareRing :segments="segments" />
	</button>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import ShareRing from "./ShareRing.vue";
import { groupColor } from "./groups";
import type { SourceRow } from "./mix";

// where an entity's energy came from as a ring, opens the details
export default defineComponent({
	name: "SourcesButton",
	components: { ShareRing },
	props: {
		rows: { type: Array as PropType<SourceRow[]>, default: () => [] },
	},
	emits: ["click"],
	computed: {
		// a site with neither solar nor battery has nothing to tell apart
		hasOwnSources(): boolean {
			return this.rows.some((row) => row.from !== "grid");
		},
		segments(): { value: number; color: string }[] {
			return this.rows.map((row) => ({ value: row.share, color: groupColor(row.from) }));
		},
	},
});
</script>

<style scoped>
/* as high as the chart type selector beside it, with its outline on hover: tells a
   mouse user the ring is a button */
.sources {
	padding: 6px;
	border: 1px solid transparent;
}
@media (hover: hover) {
	.sources:hover {
		border-color: var(--bs-border-color);
	}
}
</style>
