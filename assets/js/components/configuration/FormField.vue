<template>
	<div class="form-group">
		<label :for="name">
			{{ label }}
			<small class="text-muted" v-if="!required"> (optional) </small>
		</label>
		<input
			type="text"
			v-if="type === 'string' && !this.enum"
			class="form-control"
			:placeholder="this.default"
			value=""
			:name="name"
			:id="name"
		/>
		<input
			type="number"
			v-if="type === 'int' || type === 'uint8'"
			class="form-control"
			style="width: 50%"
			:placeholder="this.default"
			:name="name"
			value=""
			:id="name"
		/>
		<select v-if="type === 'string' && this.enum" class="custom-select" :name="name" :id="name">
			<option v-if="!required" value="">- bitte wählen -</option>
			<option :key="value" :value="value" v-for="value in this.enum">
				{{ value }}
			</option>
		</select>
	</div>
</template>

<script>
export default {
	name: "FormField",
	props: {
		type: String,
		enum: Array,
		name: String,
		value: String,
		default: String,
		label: String,
		required: Boolean,
	},
};
</script>
