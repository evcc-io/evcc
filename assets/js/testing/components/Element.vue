<template>
	<div class="row">
		<div class="col">
			<!-- <h4 class="my-4">{{ label }} ({{ type }})</h4> -->
			<form>
				<Field
					v-for="(field, idx) in fields"
					v-bind="field"
					:key="type + idx"
					:ref="type + idx"
					v-on:updated="clearStatus"
				></Field>
				<button type="submit" class="btn btn-primary btn-small" @click="test">Test</button
				>{{ this.error }}
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
import Field from "./Field";

export default {
	name: "Element",
	components: { Field },
	props: {
		configclass: String,
		type: String,
		label: String,
		fields: Array,
	},
	data: function () {
		return {
			error: null,
			result: Object,
		};
	},
	methods: {
		values: function () {
			let json = {
				Type: this.type,
			};

			console.log("element");
			console.log(this);
			console.log(this.$refs);

			for (var idx in this.$refs) {
				if (this.$refs[idx].length) {
					let field = this.$refs[idx][0];
					console.log(field);
					let val = field.values();
					if (val !== undefined) {
						json[field.name] = val;
					}
				}
			}

			console.log(json);
			return json;
		},
		test: async function (e) {
			e.preventDefault();

			const json = this.values();
			try {
				let res = await axios.post("config/test/" + this.configclass, json);
				this.result = res.data;
				this.error = null;
			} catch (e) {
				console.log(e);
				if (e.response) {
					this.error = e.response.data;
				}
			}
		},
		clearStatus: function () {
			console.log("clearStatus");
			this.result = null;
			this.error = null;
		},
	},
};
</script>
