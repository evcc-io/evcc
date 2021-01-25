<template>
	<div>
		<div class="row my-3">
			<div class="col">
				{{ list }}
				<select class="form-control" v-model="type">
					<option
						v-for="(cfg, idx) in types"
						:key="cfg.type"
						:value="idx"
						:selected="idx == type"
					>
						{{ cfg.label }}
					</option>
				</select>
			</div>
		</div>

		<FieldSet v-bind="types[type]" :klass="klass" :plugins="plugins"></FieldSet>
	</div>
</template>

<script>
import axios from "axios";
import FieldSet from "./FieldSet";

export default {
	name: "TypeSelector",
	components: { FieldSet },
	props: {
		klass: String,
		plugins: Array,
	},
	data: function () {
		return {
			types: [],
			type: 0,
		};
	},
	watch: {
		plugins: function () {
			if (this.klass == "plugin") {
				this.types = this.plugins;
				return;
			}
		},
	},
	computed: {
		list: function () {
			return this.types.map((v) => {
				return v.type;
			});
		},
	},
	mounted: async function () {
		try {
			if (this.klass != "plugin") {
				this.types = (await axios.get("/config/types/" + this.klass)).data;
			}
		} catch (e) {
			window.toasts.error(e);
		}
	},
};
</script>
