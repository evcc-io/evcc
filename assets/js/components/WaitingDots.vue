<template>
	<span
		class="dots-animation position-relative d-inline-block lh-1"
		:class="`orientation-${orientation} direction-${direction}`"
	>
		<fa-icon class="dots-back lh-1" :icon="icon"></fa-icon
		><fa-icon class="dots-front lh-1 position-absolute start-0 top-0" :icon="icon"></fa-icon>
	</span>
</template>

<script>
export default {
	name: "WaitingDots",
	props: {
		orientation: { type: String, default: "horizontal" },
		direction: { type: String, default: "up" },
	},
	computed: {
		icon() {
			return this.orientation === "horizontal" ? "ellipsis-h" : "ellipsis-v";
		},
	},
};
</script>

<style scoped>
.dots-animation {
	vertical-align: baseline;
}
.dots-animation .dots-front,
.dots-animation .dots-back {
	display: block;
}

.dots-animation .dots-back {
	opacity: 0.25;
}
.dots-animation .dots-front {
	opacity: 1;
}
.orientation-horizontal.direction-up .dots-front {
	animation: dots-horizontal 2s infinite normal steps(4, jump-none);
}
.orientation-horizontal.direction-down .dots-front {
	animation: dots-horizontal 2s infinite reverse steps(4, jump-none);
}
.orientation-vertical.direction-up .dots-front {
	animation: dots-vertical 2s infinite normal steps(4, jump-none);
}
.orientation-vertical.direction-down .dots-front {
	animation: dots-vertical 2s infinite reverse steps(4, jump-none);
}

@keyframes dots-horizontal {
	25% {
		clip-path: inset(0 100% 0 0);
	}
	75% {
		clip-path: inset(0 0 0 0);
	}
}

@keyframes dots-vertical {
	25% {
		clip-path: inset(100% 0 0 0);
	}
	75% {
		clip-path: inset(0 0 0 0);
	}
}
</style>
