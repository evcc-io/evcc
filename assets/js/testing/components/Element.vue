<template>
	<div class="form-row">
		<div class="col">
			<!-- <h4 class="my-4">{{ label }} ({{ type }})</h4> -->
			<form>
				<Field
					v-for="(field, idx) in fields"
					v-bind="field"
					:key="label + idx"
					:ref="label + idx"
					:plugins="plugins"
					v-on:updated="clearStatus"
				></Field>

				<button type="submit" class="btn btn-primary btn-small" @click="test">Test</button>

				{{ this.error }}

				<ul v-if="Object.keys(result).length">
					<li v-for="(val, idx) in result" :key="idx">
						{{ idx }}: <span v-if="val.error">{{ val.error }}</span
						><span v-else>{{ val.value }}</span>
					</li>
				</ul>
			</form>
		</div>
	</div>
</template>

<script>
import axios from "axios";
// import Field from "./Field";

export default {
	name: "Element",
	components: { Field: () => import("./Field") },
	props: {
		configclass: String,
		type: String,
		label: String,
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
			let json = {
				Type: this.type,
			};

			for (var idx in this.$refs) {
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
				let res = await axios.post("config/test/" + this.configclass, json);
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
