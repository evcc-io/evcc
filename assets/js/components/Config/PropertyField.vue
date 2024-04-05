<template>
	<div v-if="unitValue" class="input-group" :class="sizeClass">
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
				:aria-label="key"
			>
				<VehicleIcon :name="key" />
			</label>
		</div>
		<div v-if="!selectMode" class="me-2 mb-2 d-flex align-items-end">
			<a :id="id" class="text-muted" href="#" @click.prevent="toggleSelectMode">change</a>
		</div>
	</div>
	<select v-else-if="select" :id="id" v-model="value" class="form-select" :class="sizeClass">
		<option v-if="!required" value="">---</option>
		<template v-for="({ key, name }, idx) in selectOptions">
			<option v-if="key !== null && name !== null" :key="key" :value="key">
				{{ name }}
			</option>
			<hr v-else :key="idx" />
		</template>
	</select>
	<textarea
		v-else-if="textarea"
		:id="id"
		v-model="value"
		class="form-control"
		:class="sizeClass"
		:type="inputType"
		:placeholder="placeholder"
		:required="required"
		rows="4"
	/>
	<input
		v-else
		:id="id"
		v-model="value"
		class="form-control"
		:class="sizeClass"
		:type="inputType"
		:step="step"
		:placeholder="placeholder"
		:required="required"
	/>
</template>

<script>
import VehicleIcon from "../VehicleIcon";

export default {
	name: "PropertyField",
	components: { VehicleIcon },
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
		validValues: { type: Array, default: () => [] },
		modelValue: [String, Number, Boolean, Object],
	},
	emits: ["update:modelValue"],
	data: () => {
		return { selectMode: false };
	},
	computed: {
		inputType() {
			if (this.masked) {
				return "password";
			}
			if (["Number", "Float"].includes(this.type)) {
				return "number";
			}
			return "text";
		},
		sizeClass() {
			if (this.size) {
				return this.size;
			}
			if (["Number", "Float", "Duration"].includes(this.type)) {
				return "w-50 w-min-200";
			}
			return "";
		},
		endAlign() {
			return ["Number", "Float", "Duration"].includes(this.type);
		},
		step() {
			if (this.type === "Float") {
				return "any";
			}
			return null;
		},
		unitValue() {
			if (this.unit) {
				return this.unit;
			}
			if (this.property === "capacity") {
				return "kWh";
			}
			return null;
		},
		icons() {
			return this.property === "icon";
		},
		textarea() {
			return ["accessToken", "refreshToken"].includes(this.property);
		},
		select() {
			return this.validValues.length > 0;
		},
		selectOptions() {
			// If the valid values are already in the correct format, return them
			if (typeof this.validValues[0] === "object") {
				return this.validValues;
			}

			// Otherwise, convert them to the correct format
			return this.validValues.map((value) => ({
				key: value,
				name: this.$t(`config.options.${this.property}.${value}`),
			}));
		},
		value: {
			get() {
				// use first option if no value is set
				if (this.selectOptions.length > 0 && !this.modelValue) {
					return this.selectOptions[0].key;
				}

				if (this.scale) {
					return this.modelValue * this.scale;
				}

				return this.modelValue;
			},
			set(value) {
				let newValue = value;

				if (this.scale) {
					newValue = value / this.scale;
				}

				this.$emit("update:modelValue", newValue);
			},
		},
	},
	methods: {
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
