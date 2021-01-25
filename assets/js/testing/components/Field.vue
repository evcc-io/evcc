<template>
	<div>
		<div class="form-row" v-if="type == 'plugin'">
			<div class="col-4 font-weight-bold">{{ label }}</div>
			<div class="col-8">
				<select class="form-control" v-model="plugin">
					<option v-if="!required" value="">- bitte wählen -</option>
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
				<FieldSet
					v-bind="plugins[plugin]"
					:klass="'plugin'"
					:plugins="plugins"
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
					<option
						v-for="(e, idx) in enums"
						:key="idx"
						:selected="idx == value"
						:value="e"
					>
						{{ e }}
					</option>
				</select>
				<textarea class="form-control" rows="5" v-else-if="this.type == 'text'"></textarea>
				<input
					class="form-control"
					:type="this.inputType"
					:name="this.name"
					v-model="checked"
					v-else-if="isBool"
				/>
				<input
					class="form-control"
					:type="this.inputType"
					:name="this.name"
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
	components: { FieldSet: () => import("./FieldSet") },
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
	// watch: {
	// 	value: function () {
	// 		this.$emit("updated");
	// 	},
	// },
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
					if (this.masked) {
						return "password";
					}
					return "text";
				case "bool":
					return "checkbox";
				default:
					return /int|float/.test(this.type) ? "number" : this.type;
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
