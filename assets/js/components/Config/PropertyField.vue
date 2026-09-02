<template>
	<div v-if="icons" class="d-flex flex-wrap">
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
				:disabled="disabled"
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
		:disabled="disabled"
	/>
	<select
		v-else-if="select"
		:id="id"
		v-model="value"
		class="form-select"
		:class="inputClasses"
		:required="required"
		:disabled="disabled"
	>
		<option v-if="!required || !modelValue" value="" :disabled="disabled">---</option>
		<template v-for="({ key, name }, idx) in selectOptions">
			<option
				v-if="key !== null && name !== null"
				:key="key"
				:value="key"
				:disabled="disabled"
			>
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
		:rows="textareaRows"
		:disabled="disabled"
	/>
	<PropertyZonesField
		v-else-if="zones"
		:id="id"
		v-model="value"
		:currency="currency"
		:valueLabel="zonesValueLabel"
	/>
	<div v-else class="d-flex" :class="sizeClass">
		<div class="position-relative flex-grow-1 shrinkable">
			<input
				:id="id"
				:value="value"
				:list="datalistId"
				:type="inputType"
				:step="step"
				:placeholder="placeholder"
				:required="required"
				:pattern="patternRegex"
				:title="patternTitle"
				:aria-describedby="unitValue ? id + '_unit' : null"
				:class="`${datalistId && serviceValues.length > 0 ? 'form-select' : 'form-control'} ${showClearButton ? 'has-clear-button' : ''} ${invalid ? 'is-invalid' : ''} ${endAlign ? 'text-end' : ''}`"
				:style="
					unitValue ? 'border-top-right-radius: 0; border-bottom-right-radius: 0' : null
				"
				:autocomplete="masked || datalistId ? 'off' : null"
				:disabled="disabled"
				@change="onFieldChange"
				@input="onFieldInput"
			/>
			<button
				v-if="showClearButton"
				type="button"
				class="form-control-clear"
				:aria-label="$t('config.general.clear')"
				:disabled="disabled"
				@click="value = ''"
			></button>
			<datalist v-if="showDatalist" :id="datalistId">
				<option v-for="v in serviceValues" :key="v" :value="v">
					{{ v }}
				</option>
			</datalist>
		</div>
		<CustomSelect
			v-if="unitSelectable"
			:id="id + '_unit_select'"
			:options="unitOptions"
			:selected="durationUnit"
			:aria-label="$t('config.form.durationUnit', { label })"
			@change="onUnitChange"
		>
			<span
				:id="id + '_unit'"
				class="input-group-text h-100"
				style="border-top-left-radius: 0; border-bottom-left-radius: 0"
				>{{ unitValue }}</span
			>
		</CustomSelect>
		<span
			v-else-if="unitValue"
			:id="id + '_unit'"
			class="input-group-text"
			style="border-top-left-radius: 0; border-bottom-left-radius: 0"
			>{{ unitValue }}</span
		>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/minus";
import VehicleIcon from "../VehicleIcon";
import SelectGroup from "../Helper/SelectGroup.vue";
import CustomSelect from "../Helper/CustomSelect.vue";
import PropertyZonesField from "./PropertyZonesField.vue";
import formatter from "@/mixins/formatter";
import parseGoDuration, {
	displayFactors,
	durationUnits,
	goDurationUnit,
	toGoDuration,
} from "@/utils/goDuration";

const NS_PER_SECOND = 1000000000;

export default {
	name: "PropertyField",
	components: { VehicleIcon, SelectGroup, CustomSelect, PropertyZonesField },
	mixins: [formatter],
	props: {
		id: String,
		property: String,
		masked: Boolean,
		placeholder: String,
		type: String,
		unit: String,
		// transitional: emit ns numbers, remove once all callers accept duration strings (loadpoint follow-up)
		legacyDuration: Boolean,
		size: String,
		scale: Number,
		required: Boolean,
		invalid: Boolean,
		disabled: Boolean,
		pattern: { type: Object, default: () => ({}) },
		choice: { type: Array, default: () => [] },
		modelValue: [String, Number, Boolean, Object],
		label: String,
		serviceValues: { type: Array, default: () => [] },
		currency: { type: String, default: "EUR" },
		rows: { type: Number },
	},
	emits: ["update:modelValue"],
	data: () => {
		return { selectMode: false, unitOverride: null };
	},
	computed: {
		patternRegex() {
			return this.pattern.Regex || null;
		},
		patternTitle() {
			const examples = this.pattern.Examples || [];
			if (!examples.length) return null;
			return examples.join(", ");
		},
		datalistId() {
			return this.serviceValues.length > 0 ? `${this.id}-datalist` : null;
		},
		showDatalist() {
			if (!this.datalistId) return false;
			const length = this.serviceValues.length;
			// no values
			if (length === 0) return false;
			// value selected, dont offer single same option again
			// Convert both to strings for comparison to handle number/string type mismatches
			const valueStr = String(this.value ?? "");
			if (
				this.value != null &&
				valueStr !== "" &&
				this.serviceValues.some((v) => String(v) === valueStr)
			) {
				return false;
			}
			return true;
		},
		showClearButton() {
			return this.datalistId && this.value;
		},
		inputType() {
			if (this.masked) {
				return "password";
			}
			if (["Int", "Float", "Duration", "PricePerKWh"].includes(this.type)) {
				return "number";
			}
			return "text";
		},
		sizeClass() {
			if (this.size) {
				return this.size;
			}
			if (["Int", "Float", "Duration", "PricePerKWh", "ChargeModes"].includes(this.type)) {
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
			return ["Int", "Float", "Duration", "PricePerKWh"].includes(this.type);
		},
		step() {
			if (this.type === "Float" || this.type === "Duration" || this.type === "PricePerKWh") {
				return "any";
			}
			return null;
		},
		unitValue() {
			if (this.type === "Duration") {
				return this.fmtDurationUnit(this.value, this.durationUnit);
			}
			if (this.pricePerKWh) {
				return this.pricePerKWhUnit(this.currency);
			}
			if (this.unit) {
				return this.unit;
			}
			return null;
		},
		useLazyBinding() {
			// avoid conversion loop issues
			return this.inputType === "number";
		},
		icons() {
			return this.property === "icon";
		},
		textarea() {
			return (
				this.rows ||
				this.array ||
				["accessToken", "refreshToken", "identifiers", "formula"].includes(this.property)
			);
		},
		textareaRows() {
			if (this.rows) return this.rows;
			const autoGrow = this.property === "formula";
			if (autoGrow) {
				const lines = (this.value ?? "").split("\n").length;
				return Math.max(1, lines);
			}
			return 4;
		},
		boolean() {
			return this.type === "Bool";
		},
		array() {
			return this.type === "List";
		},
		zones() {
			return this.type === "Zones";
		},
		zonesValueLabel() {
			return this.property === "chargesZones"
				? this.$t("config.tariff.zones.charge")
				: this.$t("config.tariff.zones.price");
		},
		pricePerKWh() {
			return this.type === "PricePerKWh";
		},
		chargeModes() {
			return this.type === "ChargeModes";
		},
		select() {
			return this.choice.length > 0 || this.chargeModes;
		},
		durationFactor() {
			return displayFactors[this.durationUnit] ?? 1;
		},
		durationUnit() {
			return this.unitOverride ?? goDurationUnit(this.modelValue) ?? this.unit ?? "second";
		},
		unitSelectable() {
			return this.type === "Duration" && !this.legacyDuration && !this.disabled;
		},
		unitOptions() {
			return durationUnits.map((value) => ({
				value,
				name: this.fmtDurationUnit(2, value),
			}));
		},
		selectOptions() {
			if (this.chargeModes) {
				return [
					{ key: "off", name: this.$t("main.mode.off") },
					{ key: "smart", name: this.$t("main.mode.smart") },
					{ key: "now", name: this.$t("main.mode.now") },
				];
			}
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
				if (this.select && this.modelValue == null) {
					return "";
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

				if (this.type === "Duration") {
					const ns =
						typeof this.modelValue === "string"
							? parseGoDuration(this.modelValue)
							: this.modelValue;
					if (typeof ns === "number") {
						return ns / this.durationFactor / NS_PER_SECOND;
					}
					return "";
				}

				if (this.pricePerKWh) {
					const value = this.modelValue * this.pricePerKWhDisplayFactor(this.currency);
					// Round to 6 decimals to eliminate floating-point errors
					return Math.round(value * 1e6) / 1e6;
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
					newValue = this.legacyDuration
						? newValue * this.durationFactor * NS_PER_SECOND
						: toGoDuration(newValue, this.durationUnit);
				}

				if (this.pricePerKWh) {
					newValue = value / this.pricePerKWhDisplayFactor(this.currency);
				}

				this.$emit("update:modelValue", newValue);
			},
		},
	},
	methods: {
		coerceValue(val) {
			if (this.inputType === "number") {
				return val === "" ? "" : Number(val);
			}
			return val;
		},
		onUnitChange(e) {
			// read display value before override changes the getter's unit
			const num = this.value;
			this.unitOverride = e.target.value;
			if (typeof num === "number") {
				this.$emit("update:modelValue", toGoDuration(num, this.unitOverride));
			}
		},
		onFieldChange(e) {
			// unparsable input (e.g. locale decimal separator mismatch)
			if (e.target.validity?.badInput) return;
			this.value = this.coerceValue(e.target.value);
		},
		onFieldInput(e) {
			if (!this.useLazyBinding) {
				this.value = this.coerceValue(e.target.value);
			}
		},
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

<style scoped>
.shrinkable {
	min-width: 0;
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
