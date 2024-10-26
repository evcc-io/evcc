<template>
	<FormRow
		:id="id"
		:optional="!Required"
		:label="Description || `[${Name}]`"
		:help="Description === Help ? undefined : Help"
		:example="Example"
	>
		<PropertyField
			:id="id"
			v-model="value"
			:masked="Mask"
			:property="Name"
			:type="Type"
			class="me-2"
			:required="Required"
			:validValues="ValidValues"
		/>
	</FormRow>
</template>

<script>
/* eslint-disable vue/prop-name-casing */
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";

export default {
	name: "PropertyEntry",
	components: { FormRow, PropertyField },
	props: {
		id: String,
		Name: String,
		Required: Boolean,
		Description: String,
		Help: String,
		Example: String,
		Type: String,
		Mask: Boolean,
		ValidValues: Array,
		modelValue: [String, Number, Boolean, Object],
	},
	emits: ["update:modelValue"],
	computed: {
		value: {
			get() {
				return this.modelValue;
			},
			set(value) {
				this.$emit("update:modelValue", value);
			},
		},
	},
};
</script>
