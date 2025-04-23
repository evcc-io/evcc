<template>
	<component :is="icon" class="icon"></component>
</template>

<script>
import _zero from "./zero.vue";
import _minus1 from "./minus-1.vue";
import _minus2 from "./minus-2.vue";
import _minus3 from "./minus-3.vue";
import _plus1 from "./plus-1.vue";
import _plus2 from "./plus-2.vue";
import _plus3 from "./plus-3.vue";

export default {
	name: "PriorityIcon",
	props: {
		prio: { type: Number, default: 0 },
	},
	computed: {
		icon() {
			// Shortened logic with direct return of components
			if (this.prio === 0) return _zero;
			if (this.prio === 1) return _plus1;
			if (this.prio === 2) return _plus2;
			if (this.prio === 3) return _plus3;
			if (this.prio === -1) return _minus1;
			if (this.prio === -2) return _minus2;
			if (this.prio === -3) return _minus3;

			// For prios beyond the range of -3 to 3
			return this.prio > 3 ? _plus3 : _minus3;
		},
	},
};
</script>

<style scoped>
.icon {
	display: block;
	width: 24px;
	height: 24px;
}
</style>
