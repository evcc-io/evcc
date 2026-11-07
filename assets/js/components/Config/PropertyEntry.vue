<template>
	<FormRow
		:id="id"
		:optional="!Required"
		:deprecated="Deprecated"
		:readonly="Readonly"
		:label="label"
		:help="help"
		:example="example"
	>
		<template v-if="Readonly">
			<input :id="id" type="text" class="form-control border" :value="value" readonly />
			<CopyLink v-if="value" :text="String(value)" />
		</template>
		<PropertyField
			v-else
			:id="id"
			v-model="value"
			:masked="Mask"
			:property="Name"
			:type="Type"
			:unit="Unit"
			:required="Required"
			:pattern="Pattern"
			:choice="Choice"
			:device-class="Class"
			:usage="Usage"
			:has-device-error="hasDeviceError"
			:service-values="serviceValues"
			:service-loading="serviceLoading"
			:label="label"
			:currency="currency"
			:default-value="Default"
		/>
	</FormRow>
</template>

<script>
/* oxlint-disable vue/prop-name-casing */
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import CopyLink from "../Helper/CopyLink.vue";
import formatter from "@/mixins/formatter";
import { goDurationToUnit, goDurationUnit } from "@/utils/goDuration";

export default {
	name: "PropertyEntry",
	components: { FormRow, PropertyField, CopyLink },
	mixins: [formatter],
	props: {
		id: String,
		Name: String,
		Required: Boolean,
		Deprecated: Boolean,
		Readonly: Boolean,
		Description: String,
		Help: String,
		Example: String,
		Default: { type: [String, Number, Boolean], default: undefined },
		Type: String,
		hasDeviceError: { type: Function, default: () => false },
		Unit: String,
		Mask: Boolean,
		Pattern: { type: Object, default: () => ({}) },
		Choice: Array,
		Class: String,
		Usage: String,
		serviceValues: Array,
		serviceLoading: Boolean,
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
			// device names are picked, not typed: the example only serves yaml docs
			if (this.Type === "Device") return undefined;
			// show duration example in its own unit, field unit is user-changeable
			const unit = this.Type === "Duration" ? goDurationUnit(this.Example) : null;
			if (!unit) return this.Example;
			const value = goDurationToUnit(this.Example, unit);
			return `${value} ${this.fmtDurationUnit(value, unit)}`;
		},
	},
};
</script>
