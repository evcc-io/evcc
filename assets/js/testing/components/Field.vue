<template>
	<div>
		<div v-if="this.type == 'struct'">
			<h5>{{ label }}</h5>
			<div class="ml-3" v-if="this.type == 'struct'">
				<Field
					v-for="(f, idx) in this.children"
					v-bind="f"
					:key="type + idx"
					:ref="idx"
				></Field>
			</div>
		</div>
		<div class="mb-3 row" v-else>
			<label :for="this.name" class="col-sm-4 col-form-label">{{ label }}</label>

			<div class="col-sm-8">
				<select
					class="form-control"
					:id="this.name"
					:name="this.name"
					v-model="value"
					v-if="isEnum"
				>
					<option v-for="(e, idx) in enums" :key="type + idx" :value="e">{{ e }}</option>
				</select>
				<input
					class="form-control"
					:type="this.inputType"
					:name="this.name"
					:value="this.default"
					v-model="value"
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
		return {
			value: this.default,
		};
	},
	computed: {
		enums: function () {
			return this.enum;
		},
		isEnum: function () {
			return typeof this.enum !== "undefined" && typeof this.enum.length;
		},
		inputType: function () {
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
	methods: {
		values: function () {
			if (this.type != "struct") {
				return this.value;
			}

			let json = {};
			for (var idx in this.$refs) {
				let field = this.$refs[idx][0];
				let val = field.values();
				if (val !== undefined) {
					json[field.name] = val;
				}
			}

			return json;
		},
	},
};
</script>
