<template>
	<button class="btn btn-sm border-0" :disabled="disabled" @click="onClick">
		<component
			:is="icon"
			size="s"
			:class="[
				iconClass,
				{ 'nudge-prev': highlight && prev, 'nudge-next': highlight && next },
			]"
		></component>
	</button>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import "@h2d2/shopicons/es/regular/angledoubleleftsmall";
import "@h2d2/shopicons/es/regular/angledoublerightsmall";

export default defineComponent({
	name: "DateNavigatorButton",
	props: {
		disabled: Boolean,
		prev: Boolean,
		next: Boolean,
		highlight: Boolean,
		onClick: { type: Function as PropType<(event: MouseEvent) => void> },
	},
	computed: {
		icon() {
			if (this.prev) {
				return "shopicon-regular-angledoubleleftsmall";
			} else if (this.next) {
				return "shopicon-regular-angledoublerightsmall";
			}
			return null;
		},
		iconClass() {
			return this.prev ? "me-1" : this.next ? "ms-1" : "";
		},
	},
});
</script>

<style scoped>
.btn,
.btn:active,
.btn:focus {
	color: inherit !important;
}
@keyframes nudge {
	0%,
	100% {
		transform: translateX(0);
	}
	35% {
		transform: translateX(var(--nudge-distance));
	}
}
.nudge-prev,
.nudge-next {
	animation: nudge var(--evcc-transition-fast) ease-out;
}
.nudge-prev {
	--nudge-distance: -4px;
}
.nudge-next {
	--nudge-distance: 4px;
}
</style>
