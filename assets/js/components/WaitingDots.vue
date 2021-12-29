<template>
	<div class="root position-relative" :class="`direction-${direction}`">
		<shopicon-options-filled class="dots dots-back"></shopicon-options-filled
		><shopicon-options-filled
			class="dots dots-front position-absolute start-0 top-0"
		></shopicon-options-filled>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/filled/options";

export default {
	name: "WaitingDots",
	props: {
		direction: { type: String, default: "up" },
	},
};
</script>

<style scoped>
.dots {
	height: 1em;
	width: 1em;
}
.dots-front,
.dots-back {
	display: block;
}

.dots-back {
	opacity: 0.25;
}
.dots-front {
	opacity: 1;
}
.direction-up .dots-front {
	animation: dots-vertical 2s infinite normal steps(4, jump-none);
}
.direction-down .dots-front {
	animation: dots-vertical 2s infinite reverse steps(4, jump-none);
}

@keyframes dots-vertical {
	25% {
		clip-path: inset(100% 0 0 0);
	}
	75% {
		clip-path: inset(0 0 0 0);
	}
}

.root >>> svg {
	display: block;
}
</style>
