<template>
	<li class="root" :class="{ attention }">
		<button
			class="d-flex align-items-center justify-content-center p-3"
			tabindex="0"
			@click="$emit('click')"
		>
			<shopicon-regular-plus class="me-1"></shopicon-regular-plus>
			{{ title }}
		</button>
	</li>
</template>

<script>
import "@h2d2/shopicons/es/regular/plus";

export default {
	name: "NewDeviceButton",
	props: {
		title: String,
		attention: Boolean,
	},
	emits: ["click"],
};
</script>

<style scoped>
.root {
	padding: 0;
	display: block;
	list-style-type: none;
	min-height: 9rem;
	border-radius: 1rem;
	border: 1px solid var(--evcc-gray-50);
	padding: 1rem 1rem 0.5rem;
	transition: border-color var(--evcc-transition-fast) linear;
}
.root:hover,
.root:focus-within {
	border-color: var(--evcc-default-text);
	color: var(--evcc-default-text);
}
button {
	border-radius: 1rem;
	background: none;
	border: none;
	height: 100%;
	width: 100%;
	color: inherit;
}
button:focus {
	outline: none;
}
.attention {
	animation: wiggle 3s cubic-bezier(0.36, 0.07, 0.19, 0.97) infinite;
}

@keyframes wiggle {
	0%,
	100% {
		transform: rotate(0deg);
	}
	5%,
	15% {
		transform: rotate(-2deg);
	}
	10%,
	20% {
		transform: rotate(2deg);
	}
	25% {
		transform: rotate(0deg);
	}
}
</style>
