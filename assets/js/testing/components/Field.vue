<template>
	<div>
		<div class="form-row" v-if="type == 'plugin'">
			<div class="col-4 font-weight-bold">{{ label }}</div>
			<div class="col-8">
				<select class="form-control" v-model="plugin">
					<option v-if="!required" value="">- bitte wählen -</option>
					<option v-for="(cfg, idx) in plugins" :key="idx" :value="idx">
						{{ cfg.label }}
					</option>
				</select>
			</div>

			<div class="col-4"></div>
			<div class="col-8">
				<FieldSet
					v-bind="plugins[plugin]"
					:plugins="plugins"
					klass="plugin"
					ref="sub"
					v-if="plugins[plugin]"
				></FieldSet>
			</div>
		</div>

		<div class="form-row" v-else-if="type == 'struct'">
			<div class="col-4">{{ label }}</div>

			<div class="col-8">
				<FieldSet :fields="children" :plugins="plugins" ref="sub"></FieldSet>
			</div>
		</div>

		<div class="form-row" v-else>
			<label :for="name" class="col-sm-4 col-form-label">{{ label }}</label>

			<div class="col-sm-8">
				<select class="form-control" :id="name" :name="name" v-model="value" v-if="isEnum">
					<option v-if="!required" value="">- bitte wählen -</option>
					<option v-for="(e, idx) in enums" :key="idx" :value="e">
						{{ e }}
					</option>
				</select>

				<textarea
					class="form-control"
					rows="5"
					v-model="value"
					v-else-if="type == 'text'"
				></textarea>

				<input
					class="form-control"
					:type="inputType"
					:name="name"
					autocomplete="current-password"
					v-model="value"
					v-else-if="type == 'password'"
				/>

				<input
					class="form-control"
					:type="inputType"
					:name="name"
					v-model="checked"
					v-else-if="isBool"
				/>

				<input class="form-control" :type="inputType" :name="name" v-model="value" v-else />
			</div>
		</div>
	</div>
</template>

<script>
export default {
	name: "Field",
	components: { FieldSet: () => import("./FieldSet") },
	props: {
		name: String,
		label: String,
		type: String,
		required: Boolean,
		default: [String, Number, Boolean],
		enum: Array,
		children: Array,
		plugins: Array,
	},
	data: function () {
		return {
			plugin: "",
			value: this.default !== false ? this.default : "",
			checked: this.isBool ? this.default : false,
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
		isSimple: function () {
			return !(this.type == "struct" || this.type == "plugin");
		},
		inputType: function () {
			switch (this.type) {
				case "string":
					return "text";
				case "bool":
					return "checkbox";
				case /int|float/.test(this.type):
					return "number";
				default:
					return this.type;
			}
		},
	},
	methods: {
		values: function () {
			if (this.isSimple) {
				return this.isBool ? this.checked : this.value;
			}
			return this.$refs.sub.values();
		},
	},
};
</script>
