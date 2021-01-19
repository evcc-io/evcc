<template>
	<div>
		<div v-if="this.typ == 'struct'">
			<h5>{{ label }}</h5>
			<div class="ml-3" v-if="this.typ == 'struct'">
				<Field v-for="(f, idx) in this.children" v-bind="f" :key="idx"></Field>
			</div>
		</div>
		<div class="mb-3 row" v-else>
			<label :for="this.name" class="col-sm-4 col-form-label">{{ label }}</label>
			<div class="col-sm-8">
				<select class="form-control" :id="this.name" :name="this.name" v-if="isEnum">
					<option v-for="(e, idx) in enums" :key="idx" :value="e">{{ e }}</option>
				</select>
				<input
					class="form-control"
					:type="this.typ"
					:name="this.name"
					:value="this.default"
					v-else
				/>
			</div>
		</div>
	</div>
</template>

<script>
export default {
	name: "Field",
	props: {
		name: String,
		label: String,
		type: String,
		default: [String, Number],
		enum: Array,
		children: Array,
	},
	data: function () {
		return {};
	},
	computed: {
		enums: function () {
			return this.enum;
		},
		isEnum: function () {
			return typeof this.enum !== "undefined" && typeof this.enum.length;
		},
		typ: function () {
			console.log(this.type);
			switch (this.type) {
				case "string":
					return "text";
				case "bool":
					return "checkbox";
				default:
					return this.type;
			}
		},
	},
};
</script>
