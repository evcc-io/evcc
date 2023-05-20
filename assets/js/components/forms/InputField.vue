<template>
	<div v-if="unit" class="input-group w-100">
		<input
			:id="id"
			v-model="value"
			:type="type"
			:placeholder="placeholder"
			:required="required"
			aria-label="unit"
			:aria-describedby="id + '_unit'"
			class="form-control"
		/>
		<span :id="id + '_unit'" class="input-group-text">{{ unit }}</span>
	</div>
	<textarea
		v-else-if="textarea"
		:id="id"
		v-model="value"
		class="form-control"
		:type="type"
		:placeholder="placeholder"
		:required="required"
		rows="4"
	/>
	<input
		v-else
		:id="id"
		v-model="value"
		class="form-control"
		:type="type"
		:placeholder="placeholder"
		:required="required"
	/>
</template>

<script>
export default {
	name: "InputField",
	props: {
		id: String,
		property: String,
		masked: Boolean,
		optional: Boolean,
		placeholder: String,
		required: Boolean,
		modelValue: [String, Number, Boolean, Object],
	},
	emits: ["update:modelValue"],
	computed: {
		type() {
			return this.masked ? "password" : "text";
		},
		unit() {
			if (this.property === "capacity") {
				return "kWh";
			}
			return null;
		},
		textarea() {
			return ["accessToken", "refreshToken"].includes(this.property);
		},
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
