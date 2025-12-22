<template>
	<div v-if="unitValue" class="input-group" :class="inputClasses">
		<input
			:id="id"
			v-model="value"
			:type="inputType"
			:step="step"
			:placeholder="placeholder"
			:required="required"
			:aria-describedby="id + '_unit'"
			class="form-control"
			:class="{ 'text-end': endAlign }"
		/>
		<span :id="id + '_unit'" class="input-group-text">{{ unitValue }}</span>
	</div>
	<div v-else-if="icons" class="d-flex flex-wrap">
		<div
			v-for="{ key } in selectOptions"
			v-show="key === value || selectMode"
			:key="key"
			class="me-2 mb-2"
		>
			<input
				:id="`icon_${key}`"
				v-model="value"
				type="radio"
				:class="selectMode ? 'btn-check' : 'd-none'"
				:name="property"
				:value="key"
				@click="toggleSelectMode"
			/>
			<label
				class="btn btn-outline-secondary"
				:class="key === value ? 'active' : ''"
				:for="`icon_${key}`"
			>
				<VehicleIcon v-if="key" :name="key" />
				<shopicon-regular-minus v-else></shopicon-regular-minus>
			</label>
		</div>
		<div v-if="!selectMode" class="me-2 mb-2 d-flex align-items-end">
			<a :id="id" class="text-muted" href="#" @click.prevent="toggleSelectMode">
				{{ $t("config.icon.change") }}
			</a>
		</div>
	</div>
	<SelectGroup
		v-else-if="boolean"
		:id="id"
		v-model="value"
		class="w-50"
		equal-width
		transparent
		:aria-label="label"
		:options="[
			{ value: false, name: $t('config.options.boolean.no') },
			{ value: true, name: $t('config.options.boolean.yes') },
		]"
	/>
	<select v-else-if="select" :id="id" v-model="value" class="form-select" :class="inputClasses">
		<option v-if="!required" value="">---</option>
		<template v-for="({ key, name }, idx) in selectOptions">
			<option v-if="key !== null && name !== null" :key="key" :value="key">
				{{ name }}
			</option>
			<option v-else :key="idx" disabled>─────</option>
		</template>
	</select>
	<textarea
		v-else-if="textarea"
		:id="id"
		v-model="value"
		class="form-control"
		:class="inputClasses"
		:type="inputType"
		:placeholder="placeholder"
		:required="required"
		rows="4"
	/>
	<div v-else class="position-relative">
		<input
			:id="id"
			v-model="value"
			:list="datalistId"
			:class="`${datalistId && serviceValues.length > 0 ? 'form-select' : 'form-control'} ${inputClasses}`"
			:type="inputType"
			:step="step"
			:placeholder="placeholder"
			:required="required"
			:autocomplete="masked || datalistId ? 'off' : null"
		/>
		<button
			v-if="showClearButton"
			type="button"
			class="form-control-clear"
			:aria-label="$t('config.general.clear')"
			@click="value = ''"
		>
			&times;
		</button>
		<datalist v-if="showDatalist" :id="datalistId">
			<option v-for="v in serviceValues" :key="v" :value="v">
				{{ v }}
			</option>
		</datalist>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/minus";
import VehicleIcon from "../VehicleIcon";
import SelectGroup from "../Helper/SelectGroup.vue";
import formatter from "@/mixins/formatter";

const NS_PER_SECOND = 1000000000;

export default {
	name: "PropertyField",
	components: { VehicleIcon, SelectGroup },
	mixins: [formatter],
	props: {
		id: String,
		property: String,
		masked: Boolean,
		placeholder: String,
		type: String,
		unit: String,
		size: String,
		scale: Number,
		required: Boolean,
		invalid: Boolean,
		choice: { type: Array, default: () => [] },
		modelValue: [String, Number, Boolean, Object],
		label: String,
		serviceValues: { type: Array, default: () => [] },
	},
	emits: ["update:modelValue"],
	data: () => {
		return { selectMode: false };
	},
	computed: {
		datalistId() {
			return this.serviceValues.length > 0 ? `${this.id}-datalist` : null;
		},
		showDatalist() {
			if (!this.datalistId) return false;
			const length = this.serviceValues.length;
			// no values
			if (length === 0) return false;
			// value selected, dont offer single same option again
			if (this.value && this.serviceValues.includes(this.value)) return false;
			return true;
		},
		showClearButton() {
			return this.datalistId && this.value;
		},
		inputType() {
			if (this.masked) {
				return "password";
			}
			if (["Int", "Float", "Duration"].includes(this.type)) {
				return "number";
			}
			return "text";
		},
		sizeClass() {
			if (this.size) {
				return this.size;
			}
			if (["Int", "Float", "Duration"].includes(this.type)) {
				return "w-50 w-min-200";
			}
			return "";
		},
		inputClasses() {
			let result = this.sizeClass;
			if (this.invalid) {
				result += " is-invalid";
			}
			if (this.showClearButton) {
				result += " has-clear-button";
			}
			return result;
		},
		endAlign() {
			return ["Int", "Float", "Duration"].includes(this.type);
		},
		step() {
			if (this.type === "Float" || this.type === "Duration") {
				return "any";
			}
			return null;
		},
		unitValue() {
			if (this.type === "Duration") {
				return this.fmtDurationUnit(this.value, this.unit);
			}
			if (this.unit) {
				return this.unit;
			}
			return null;
		},
		icons() {
			return this.property === "icon";
		},
		textarea() {
			return ["accessToken", "refreshToken", "identifiers"].includes(this.property);
		},
		boolean() {
			return this.type === "Bool";
		},
		array() {
			return this.type === "List";
		},
		select() {
			return this.choice.length > 0;
		},
		durationFactor() {
			return this.unit === "minute" ? 60 : 1;
		},
		selectOptions() {
			// If the valid values are already in the correct format, return them
			if (typeof this.choice[0] === "object") {
				return this.choice;
			}

			let values = [...this.choice];

			if (this.icons && !this.required) {
				values = ["", ...values];
			}

			// Otherwise, convert them to the correct format
			return values.map((value) => ({
				key: value,
				name: this.getOptionName(value),
			}));
		},
		value: {
			get() {
				// use first option if no value is set
				if (this.selectOptions.length > 0 && !this.modelValue) {
					return this.required ? this.selectOptions[0].key : "";
				}

				if (this.scale) {
					return this.modelValue * this.scale;
				}

				if (this.boolean) {
					return this.modelValue === "true" || this.modelValue === true;
				}

				if (this.array) {
					return Array.isArray(this.modelValue) ? this.modelValue.join("\n") : "";
				}

				if (this.type === "Duration" && typeof this.modelValue === "number") {
					return this.modelValue / this.durationFactor / NS_PER_SECOND;
				}

				return this.modelValue;
			},
			set(value) {
				let newValue = value;

				if (this.scale) {
					newValue = value / this.scale;
				}

				if (this.array) {
					newValue = value ? value.split("\n") : [];
				}

				if (this.type === "Duration" && typeof newValue === "number") {
					newValue = newValue * this.durationFactor * NS_PER_SECOND;
				}

				this.$emit("update:modelValue", newValue);
			},
		},
	},
	methods: {
		getOptionName(value) {
			const translationKey = `config.options.${this.property}.${value || "none"}`;
			return this.$te(translationKey) ? this.$t(translationKey) : value;
		},
		toggleSelectMode() {
			this.$nextTick(() => {
				this.selectMode = !this.selectMode;
			});
		},
	},
};
</script>

<style>
input[type="number"] {
	appearance: textfield;
}
input[type="number"]::-webkit-outer-spin-button,
input[type="number"]::-webkit-inner-spin-button {
	-webkit-appearance: none;
	margin: 0;
}
.w-min-100 {
	min-width: min(100px, 100%);
}
.w-min-150 {
	min-width: min(150px, 100%);
}
.w-min-200 {
	min-width: min(200px, 100%);
}
</style>
