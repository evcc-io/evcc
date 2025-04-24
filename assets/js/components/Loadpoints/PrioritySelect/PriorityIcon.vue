<template>
	<div v-if="prio > 3" class="extra extra--plus">
		<Plus1 class="icon" />
		<div class="text">{{ prio }}</div>
	</div>
	<div v-else-if="prio < -3" class="extra extra--minus">
		<Minus1 class="icon" />
		<div class="text">{{ Math.abs(prio) }}</div>
	</div>
	<component :is="icon" v-else-if="icon" class="icon"></component>
</template>

<script>
import Minus1 from "./Minus1.vue";
import Minus2 from "./Minus2.vue";
import Minus3 from "./Minus3.vue";
import Plus1 from "./Plus1.vue";
import Plus2 from "./Plus2.vue";
import Plus3 from "./Plus3.vue";

export default {
	name: "PriorityIcon",
	components: { Minus1, Plus1 },
	props: {
		prio: { type: Number, default: 0 },
	},
	computed: {
		icon() {
			if (this.prio === 0) return Zero;
			if (this.prio === 1) return Plus1;
			if (this.prio === 2) return Plus2;
			if (this.prio === 3) return Plus3;
			if (this.prio === -1) return Minus1;
			if (this.prio === -2) return Minus2;
			if (this.prio === -3) return Minus3;
			return null;
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
.extra {
	position: relative;
	display: block;
	width: 24px;
	height: 24px;
}
.extra .text {
	font-size: 0.85rem;
	font-weight: bold;
	position: absolute;
	left: 0;
	right: 0;
	text-align: center;
}
.extra--plus .icon {
	transform: translate(0, -8px);
}
.extra--plus .text {
	bottom: 0;
}
.extra--minus .icon {
	transform: translate(0, 8px);
}
.extra--minus .text {
	top: 0;
}
</style>
