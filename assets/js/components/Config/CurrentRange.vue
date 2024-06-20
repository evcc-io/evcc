<template>
	<div class="root">
		<Slider
			v-model="range"
			range
			:step="step"
			:pt="{
				root: 'slider',
				range: 'range',
				startHandler: 'handle',
				endHandler: 'handle',
			}"
			:min="0"
			:max="32"
		/>
		<span class="form-text evcc-gray d-flex justify-content-between">
			<span class="d-block">min {{ min }} A, max {{ max }} A</span>
			<a href="#" class="d-block evcc-default-text ms-2" @click.prevent="range = [6, 16]">
				use defaults
			</a>
		</span>
	</div>
</template>

<script>
import Slider from "primevue/slider";

export default {
	name: "CurrentRange",
	components: { Slider },
	props: {
		min: Number,
		max: Number,
	},
	computed: {
		step() {
			return Math.pow(10, -this.decimals);
		},
		decimals() {
			const min = Math.min(this.min, this.max);
			if (min < 0.5) return 2;
			if (min < 5) return 1;
			return 0;
		},
		range: {
			get() {
				return [this.min, this.max];
			},
			set([min, max]) {
				// flip values if necessary
				if (min > max) [min, max] = [max, min];

				// ensure precision
				min = parseFloat(min.toFixed(this.decimals));
				max = parseFloat(max.toFixed(this.decimals));

				this.$emit("update:min", min);
				this.$emit("update:max", max);
			},
		},
	},
};
</script>
<style scoped>
.root {
	padding: 0.7rem 0 0;
}
.values {
	display: block;
}
.slider {
	position: relative;
	background-color: var(--bs-gray-300);
	height: 4px;
	border-radius: 2px;
	margin-bottom: 1rem;
}

:deep(.range) {
	background-color: var(--evcc-dark-green);
	top: 0;
	height: 100%;
	display: block;
	border-radius: 2px;
}
:deep(.handle) {
	top: 50%;
	margin-top: -12px;
	margin-left: -12px;
	cursor: grab;
	touch-action: none;
	display: flex;
	justify-content: center;
	align-items: center;
	height: 24px;
	width: 24px;
	background-color: var(--bs-gray-300);
	border: 0 none;
	border-radius: 50%;
	transition-property: background, color, border-color, box-shadow, outline-color;
	transition-duration: var(--evcc-transition-fast);
	outline-color: transparent;
}
:deep(.handle::before) {
	content: "";
	width: 18px;
	height: 18px;
	display: block;
	background-color: var(--evcc-box);
	border-radius: 50%;
	box-shadow:
		0px 0.5px 0px 0px rgba(0, 0, 0, 0.08),
		0px 1px 1px 0px rgba(0, 0, 0, 0.14);
}
:deep(.handle:hover) {
	background-color: var(--evcc-dark-green);
}
:deep(.handle:focus) {
	background-color: var(--evcc-dark-green);
	outline: 1px solid var(--evcc-darkest-green);
	outline-offset: 0;
}
</style>
