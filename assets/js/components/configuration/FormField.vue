<template>
	<input type="hidden" v-if="inputType === 'hidden'" :name="name" value="" />
	<div class="form-group" v-else>
		<label :for="name">
			{{ label }}
			<small class="text-muted" v-if="!required"> (optional) </small>
		</label>
		<input
			type="text"
			v-if="inputType === 'text'"
			class="form-control"
			:placeholder="this.default"
			value=""
			:name="name"
			:id="name"
		/>
		<input
			type="password"
			v-if="inputType === 'password'"
			class="form-control"
			placeholder="********"
			:name="name"
			:id="name"
		/>
		<input
			type="number"
			v-if="inputType === 'number'"
			class="form-control"
			style="width: 50%"
			:placeholder="this.default"
			:name="name"
			value=""
			:id="name"
		/>
		<select v-if="inputType === 'select'" class="custom-select" :name="name" :id="name">
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
		name: String,
		type: String,
		required: Boolean,
		hidden: Boolean,
		masked: Boolean,
		label: String,
		enum: Array,
		default: String,
	},
	computed: {
		inputType: function () {
			console.log(this);
			if (this.hidden) return "hidden";
			if (this.enum) return "select";
			if (this.masked) return "password";
			if (this.type === "int" || this.type === "uint8") return "number";
			return "text";
		},
	},
};
</script>
