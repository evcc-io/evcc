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
			:masked="Mask"
			:property="Name"
			:type="Type"
			:unit="Unit"
			:required="Required"
			:pattern="Pattern"
			:choice="Choice"
			:service-values="serviceValues"
			:label="label"
			:currency="currency"
		/>
	</FormRow>
</template>

<script>
/* oxlint-disable vue/prop-name-casing */
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import formatter from "@/mixins/formatter";
import { goDurationToUnit, goDurationUnit } from "@/utils/goDuration";

export default {
	name: "PropertyEntry",
	components: { FormRow, PropertyField },
	mixins: [formatter],
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
		Pattern: { type: Object, default: () => ({}) },
		Choice: Array,
		serviceValues: Array,
		modelValue: [String, Number, Boolean, Object],
		currency: { type: String, default: "EUR" },
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
			// show duration example in its own unit, field unit is user-changeable
			const unit = this.Type === "Duration" ? goDurationUnit(this.Example) : null;
			if (!unit) return this.Example;
			const value = goDurationToUnit(this.Example, unit);
			return `${value} ${this.fmtDurationUnit(value, unit)}`;
		},
	},
};
</script>
