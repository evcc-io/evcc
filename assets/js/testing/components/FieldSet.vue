<template>
	<div class="form-row">
		<div class="col">
			<form>
				<Field
					v-for="field in fields"
					:key="fields[0].default + field.name"
					:ref="field.name"
					v-bind="field"
					:plugins="plugins"
					v-on:updated="clearStatus"
				></Field>

				<template v-if="klass">
					<div class="row my-3" v-if="error">
						<div class="col-4 text-danger">Fehler:</div>
						<div class="col-8 text-danger">
							{{ error }}
						</div>
					</div>
					<button type="submit" class="btn btn-primary btn-small" @click="test">
						Test
					</button>

					<ul v-if="Object.keys(result).length">
						<li v-for="(val, idx) in result" :key="idx">
							{{ idx }}: <span v-if="val.error">{{ val.error }}</span
							><span v-else>{{ val.value }}</span>
						</li>
					</ul>
				</template>
			</form>
		</div>
	</div>
</template>

<script>
import axios from "axios";
import Field from "./Field";

export default {
	name: "FieldSet",
	components: { Field },
	props: {
		klass: String,
		fields: Array,
		plugins: Array,
	},
	data: function () {
		return {
			error: "",
			result: {},
		};
	},
	methods: {
		values: function () {
			let json = {};

			for (let idx in this.$refs) {
				if (this.$refs[idx].length) {
					let field = this.$refs[idx][0];

					let val = field.values();
					if (val !== undefined) {
						json[field.name] = val;
					}
				}
			}

			return json;
		},
		test: async function (e) {
			e.preventDefault();
			this.clearStatus();

			const json = this.values();
			try {
				let res = await axios.post("config/test/" + this.klass, json);
				this.result = res.data;
			} catch (e) {
				if (e.response) {
					this.error = e.response.data;
				}
			}
		},
		clearStatus: function () {
			this.result = {};
			this.error = "";
		},
	},
};
</script>
