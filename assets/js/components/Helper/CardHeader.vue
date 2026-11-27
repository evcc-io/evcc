<template>
	<div class="header d-flex align-items-center flex-wrap column-gap-3 row-gap-3">
		<div class="lead d-flex align-items-center gap-2">
			<h3
				class="title fw-normal m-0 d-flex gap-2 align-items-baseline"
				:class="{ 'fs-6': small }"
			>
				<template v-if="parent">
					<component
						:is="parentStatic ? 'span' : 'button'"
						:type="parentStatic ? undefined : 'button'"
						class="parent d-flex gap-2 align-items-center align-self-center p-0 border-0 bg-transparent text-nowrap"
						:aria-label="parentStatic ? undefined : parent"
						@click="back"
					>
						<span v-if="$slots['icon']" class="icon d-flex"><slot name="icon" /></span>
						<span :class="{ 'd-none d-lg-block': $slots['icon'] }">{{ parent }}</span>
					</component>
					<span class="subtitle" aria-hidden="true">›</span>
				</template>
				<span v-else-if="$slots['icon']" class="icon d-flex"><slot name="icon" /></span>
				<span v-if="$slots['dot']" class="d-flex align-self-center flex-shrink-0"
					><slot name="dot"
				/></span>
				<span
					class="d-block no-wrap text-truncate"
					:class="{ 'flex-shrink-0': subtitle }"
					>{{ title }}</span
				>
				<small v-if="subtitle" class="d-block no-wrap text-truncate subtitle">{{
					subtitle
				}}</small>
			</h3>
			<slot name="nav" />
		</div>
		<div v-if="$slots['stats']" class="figures d-flex align-items-center gap-3 gap-lg-4">
			<slot name="stats" />
		</div>
		<div v-if="$slots['actions']" class="ms-auto d-flex align-items-center gap-2">
			<slot name="actions" />
		</div>
		<slot name="ring" />
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

// title row of a card. Left what it is: icon, title with a `dot` before the name, a
// subtitle and `nav`. Right the numbers (`stats`), the `actions` and last the `ring`.
// Below lg the numbers take a row of their own. `parent` is a crumb before the title
// that leads back, the icon alone below lg
export default defineComponent({
	name: "CardHeader",
	props: {
		title: { type: String, default: "" },
		subtitle: { type: String, default: "" },
		parent: { type: String, default: "" },
		// the crumb leads nowhere: the title is the group's only member
		parentStatic: Boolean,
		// a subhead inside a card rather than the card title
		small: Boolean,
	},
	emits: ["parent"],
	methods: {
		back() {
			if (!this.parentStatic) this.$emit("parent");
		},
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

.icon {
	width: 24px;
	height: 24px;
	align-self: center;
}
.subtitle,
.parent {
	color: var(--evcc-gray);
}
.parent {
	font: inherit;
}
/* gives way long before the title does */
small.subtitle {
	flex-shrink: 100;
}
button.parent:hover {
	color: var(--evcc-default-text);
}
/* takes the room the numbers leave, the title truncates before the row wraps */
/* as high as an icon toggle, so titles sit at the same offset with or without one */
/* shrinks by its truncating title, not by clipping: the color dot's ring needs the room */
.title {
	min-width: 0;
}
.lead {
	flex: 1 1 12rem;
	min-width: 0;
	min-height: 42px;
}
/* on a row of their own the numbers can ask how wide it is */
.figures {
	order: 1;
	flex-basis: 100%;
	container-type: inline-size;
}
@media (--lg-and-up) {
	.figures {
		container-type: normal;
		order: 0;
		flex-basis: auto;
	}
}
</style>
