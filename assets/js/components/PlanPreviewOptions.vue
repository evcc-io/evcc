<template>
	<label
		class="position-relative d-block"
		:for="dropdownId"
		role="button"
		data-testid="change-plan-preview"
	>
		<select :id="dropdownId" :value="selectedPlan" class="custom-select" @change="change">
			<option
				v-for="{ planId, title } in planOptions"
				:key="planId"
				:value="title"
				:selected="title === selectedPlan"
			>
				{{ title }}
			</option>
		</select>
		<slot></slot>
	</label>
</template>

<script>
export default {
	name: "PlanPreviewOptions",
	props: {
		planOptions: Array,
		selectedPlan: String,
	},
	emits: ["change-preview-plan"],
	computed: {
		dropdownId() {
			return `planPreviewDropdown`;
		},
	},
	methods: {
		change(event) {
			const selectedTitle = event.target.value;
			const selectedPlanOption = this.planOptions.find(
				(option) => option.title === selectedTitle
			);

			this.$emit("change-preview-plan", {
				planId: selectedPlanOption.planId,
				value: selectedTitle,
			});
		},
	},
};
</script>
<style scoped>
.custom-select {
	left: 0;
	top: 0;
	bottom: 0;
	width: 100%;
	cursor: pointer;
	position: absolute;
	opacity: 0;
	-webkit-appearance: menulist-button;
}
</style>
