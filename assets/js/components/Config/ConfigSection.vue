<template>
	<section :id="slug">
		<template v-if="!mobile">
			<h2 class="my-4 mt-5">{{ title }}</h2>
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
/* no ring on programmatic focus */
.detail-panel {
	outline: none;
}
</style>
