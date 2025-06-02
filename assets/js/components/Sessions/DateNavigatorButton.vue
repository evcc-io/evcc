<template>
	<button class="btn btn-sm border-0" :disabled="disabled" @click="onClick">
		<component :is="icon" size="s" :class="iconClass"></component>
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
</style>
