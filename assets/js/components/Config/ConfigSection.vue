<template>
	<section :id="slug">
		<template v-if="!mobile">
			<h2 class="my-4 mt-5 d-flex align-items-center justify-content-between gap-2">
				{{ title }}
				<span class="actions-pull-out d-flex"><slot name="actions" /></span>
			</h2>
			<slot />
		</template>
		<Transition v-else name="fade-swap-right" @after-enter="focusPanel">
			<div
				v-if="active"
				ref="panel"
				class="detail-panel my-4"
				role="region"
				:aria-label="title"
				tabindex="-1"
				:data-testid="`section-detail-${slug}`"
			>
				<div
					v-if="$slots['actions']"
					class="actions-pull-out d-flex justify-content-end mb-3"
				>
					<slot name="actions" />
				</div>
				<slot />
			</div>
		</Transition>
	</section>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "ConfigSection",
	props: {
		slug: { type: String, required: true },
		title: { type: String, required: true },
		mobile: Boolean,
		active: Boolean,
	},
	methods: {
		focusPanel() {
			(this.$refs["panel"] as HTMLElement | undefined)?.focus({ preventScroll: true });
		},
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

/* no ring on programmatic focus */
.detail-panel {
	outline: none;
}

/* align section actions with the box-pull-out card edge */
.actions-pull-out {
	margin-right: -1rem;
}
@media (--sm-and-up) {
	.actions-pull-out {
		margin-right: -1.5rem;
	}
}
</style>
