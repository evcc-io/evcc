<template>
	<div class="d-flex align-items-center gap-2">
		<shopicon-regular-powersupply
			class="solar-share-icon flex-shrink-0"
		></shopicon-regular-powersupply>
		<input
			:id="id"
			:value="modelValue"
			type="range"
			class="form-range"
			min="0"
			max="100"
			step="10"
			:disabled="disabled"
			@input="onInput"
			@change="$emit('change')"
		/>
		<shopicon-regular-sun class="solar-share-icon flex-shrink-0"></shopicon-regular-sun>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/sun";

export default defineComponent({
	name: "SolarShareSlider",
	props: {
		id: String,
		modelValue: { type: Number, default: 100 },
		disabled: Boolean,
	},
	emits: ["update:modelValue", "change"],
	methods: {
		onInput(e: Event) {
			this.$emit("update:modelValue", Number((e.target as HTMLInputElement).value));
		},
	},
});
</script>

<style scoped>
.solar-share-icon {
	width: 24px;
}
</style>
