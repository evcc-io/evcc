<template>
	<div>
		<div class="form-row" v-if="type == 'struct'">
			<div class="col-4">{{ label }}</div>

			<div class="col-8">
				<Field
					v-for="(f, idx) in children"
					v-bind="f"
					:key="name + idx"
					:ref="name + idx"
				></Field>
			</div>
		</div>
		<div class="form-row" v-else-if="type == 'plugin'">
			<div class="col-4 font-weight-bold">{{ label }}</div>
			<div class="col-8">
				<select class="form-control" v-model="plugin">
					<option value="">- bitte wählen -</option>
					<option
						v-for="(cfg, idx) in plugins"
						:key="idx"
						:value="idx"
						:selected="idx == plugin"
					>
						{{ cfg.label }}
					</option>
				</select>
			</div>

			<div class="col-4"></div>
			<div class="col-8">
				<Element
					v-bind="plugins[plugin]"
					:configclass="'plugin'"
					:plugins="plugins"
					:ref="name"
				></Element>
			</div>
		</div>
		<div class="form-row" v-else>
			<label :for="this.name" class="col-sm-4 col-form-label">{{ label }}</label>

			<div class="col-sm-8">
				<select
					class="form-control"
					:id="this.name"
					:name="this.name"
					v-model="value"
					v-if="isEnum"
				>
					<option v-if="!required" value="">- bitte wählen -</option>
					<option v-for="(e, idx) in enums" :key="idx" :value="e">{{ e }}</option>
				</select>
				<input
					class="form-control"
					:type="this.inputType"
					:name="this.name"
					:value="this.default"
					v-model="checked"
					v-else-if="isBool"
				/>
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
import Element from "./Element";

export default {
	name: "Field",
	components: { Element },
	props: {
		name: String,
		label: String,
		type: String,
		masked: Boolean,
		required: Boolean,
		default: [String, Number],
		enum: Array,
		children: Array,
		plugins: Array,
	},
	data: function () {
		return {
			value: this.default,
			checked: false,
			plugin: 0,
		};
	},
	watch: {
		value: function () {
			this.$emit("updated");
		},
	},
	computed: {
		enums: function () {
			return this.enum;
		},
		isEnum: function () {
			return typeof this.enum !== "undefined" && typeof this.enum.length;
		},
		isBool: function () {
			return this.type == "bool";
		},
		inputType: function () {
			switch (this.type) {
				case "string":
					if (this.masked) {
						return "password";
					}
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
			if (this.type !== "struct") {
				if (this.isBool) {
					return this.checked;
				}
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
