<template>
	<component :is="icon" :class="`icon icon--${size}`"></component>
</template>

<script>
import _1 from "./1.vue";
import _2 from "./2.vue";
import _3 from "./3.vue";
import _4 from "./4.vue";
import _5 from "./5.vue";
import _6 from "./6.vue";
import _7 from "./7.vue";
import _8 from "./8.vue";
import _9 from "./9.vue";
import Plus from "./Plus.vue";

const icons = {
	_1,
	_2,
	_3,
	_4,
	_5,
	_6,
	_7,
	_8,
	_9,
};

export default {
	name: "MultiIcon",
	props: {
		count: { type: Number },
		size: { type: String, default: "s" },
	},
	computed: {
		icon: function () {
			return this.count > 9 ? Plus : icons[`_${this.count}`];
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
.icon--m {
	width: 32px;
	height: 32px;
}
.icon--l {
	width: 48px;
	height: 48px;
}
.icon--xl {
	width: 64px;
	height: 64px;
}
</style>
