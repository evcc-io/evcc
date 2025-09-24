<template>
	<FormRow
		:id="id"
		:optional="!Required"
		:deprecated="Deprecated"
		:label="label"
		:help="help"
		:example="example"
	>
		<PropertyField
			:id="id"
			v-model="value"
			class="me-2"
			:masked="Mask"
			:property="Name"
			:type="Type"
			:unit="Unit"
			:required="Required"
			:choice="Choice"
			:label="label"
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
		Deprecated: Boolean,
		Description: String,
		Help: String,
		Example: String,
		Type: String,
		Unit: String,
		Mask: Boolean,
		Choice: Array,
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
		label() {
			return this.Description || `[${this.Name}]`;
		},
		help() {
			return this.Description === this.Help ? undefined : this.Help;
		},
		example() {
			// hide example text since config ui doesnt use go duration format (e.g. 5m)
			return this.Type === "Duration" ? undefined : this.Example;
		},
	},
};
</script>
