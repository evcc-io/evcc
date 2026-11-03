<template>
	<div class="header d-flex align-items-center gap-2 flex-wrap">
		<slot name="prefix" />
		<h3
			class="fw-normal m-0 d-flex gap-2 flex-wrap align-items-baseline overflow-hidden"
			:class="{ 'fs-6': small }"
		>
			<span v-if="$slots['icon']" class="icon d-flex"><slot name="icon" /></span>
			<span class="d-block no-wrap text-truncate">{{ title }}</span>
			<small v-if="subtitle" class="d-block no-wrap text-truncate subtitle">{{
				subtitle
			}}</small>
		</h3>
		<slot name="nav" />
		<div v-if="$slots['actions']" class="ms-auto d-flex align-items-center gap-2">
			<slot name="actions" />
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

// title row of a card: an optional icon before the title, a subtitle beside it,
// `prefix` before the heading, `nav` after it and `actions` on the right
export default defineComponent({
	name: "CardHeader",
	props: {
		title: { type: String, default: "" },
		subtitle: { type: String, default: "" },
		// a subhead inside a card rather than the card title
		small: Boolean,
	},
});
</script>

<style scoped>
/* as high as an icon toggle in the actions, so titles sit at the same offset with or without one */
.header {
	min-height: 42px;
}
/* lifted onto the title's baseline */
.icon {
	width: 24px;
	height: 24px;
	position: relative;
	top: -0.1em;
}
.subtitle {
	color: var(--evcc-gray);
}
</style>
