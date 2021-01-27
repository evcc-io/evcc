<template>
	<div class="form-row">
		<label :for="name" class="col-4 col-form-label">{{ label }}</label>

		<div class="col-8" v-if="type == 'plugin'">
			<select class="form-control" v-model="plugin">
				<option v-if="!required" value="">- bitte wählen -</option>
				<option v-for="(cfg, idx) in plugins" :key="idx" :value="idx">
					{{ cfg.label }}
				</option>
			</select>

			<FieldSet
				v-bind="plugins[plugin]"
				:plugins="plugins"
				klass="plugin"
				ref="sub"
				v-if="plugins[plugin]"
			></FieldSet>
		</div>

		<div class="col-8" v-else-if="type == 'slice'">
			<FieldSet :fields="sliceFields" :plugins="plugins" klass="plugin" ref="sub"></FieldSet>
		</div>

		<div class="col-8" v-else-if="type == 'struct'">
			<FieldSet :fields="children" :plugins="plugins" ref="sub"></FieldSet>
		</div>

		<div class="col-8" v-else>
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

			<input
				class="form-control"
				:type="inputType"
				:step="inputStep"
				:name="name"
				v-model="value"
				v-else
			/>
		</div>
	</div>
</template>

<script>
export default {
	name: "Field",
	components: { FieldSet: () => import("./FieldSet") },
	props: {
		type: String,
		name: String,
		label: String,
		length: Number,
		required: Boolean,
		enum: Array,
		default: [String, Number, Boolean],
		children: Array,
		plugins: Array,
	},
	data: function () {
		let defVal = this.default !== false ? this.default : "";
		if (this.enum && this.enum.length && this.required && defVal === "") {
			defVal = this.enum[0];
		}
		return {
			plugin: "",
			value: defVal,
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
			return !(this.type == "struct" || this.type == "slice" || this.type == "plugin");
		},
		inputType: function () {
			switch (this.type) {
				case "string":
					return "text";
				case "bool":
					return "checkbox";
				case "duration":
					return "text"; // number?
				default:
					if (/int|float/.test(this.type)) {
						return "number";
					}
					return this.type;
			}
		},
		inputStep: function () {
			return this.inputType == "number" ? "any" : "";
		},
		sliceFields: function () {
			let res = [];
			let max = this.length ? this.length : 1;

			for (let i = 0; i < max; i++) {
				res.push({
					type: "plugin",
					label: "" + (res.length + 1),
					name: "" + res.length,
				});
			}
			return res;
		},
	},
	methods: {
		values: function () {
			if (this.type == "duration") {
				return this.value ? this.value : 0;
			}
			if (this.isSimple) {
				return this.isBool ? this.checked : this.value;
			}

			if (this.type == "slice") {
				let res = [];
				let values = this.$refs.sub.values();
				for (let val in values) {
					res.push(values[val]);
				}

				let populated = res.reduce((acc, val) => {
					return acc || val !== null;
				});
				if (this.required || populated) {
					return res;
				}

				// not required and empty
				return null;
			}

			// struct
			let sub = this.$refs.sub;
			if (sub === undefined) {
				return null;
			}

			return sub.values();
		},
	},
};
</script>
