<template>
	<div>
		<div class="row my-3">
			<div class="col">
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

		<FieldSet v-bind="types[type]" :klass="klass" :plugins="plugins" ref="fields"></FieldSet>
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
		type: function () {
			this.$refs.fields.clearStatus();
		},
	},
	mounted: async function () {
		try {
			this.types = (await axios.get("/config/types/" + this.klass)).data;
		} catch (e) {
			window.toasts.error(e);
		}
	},
};
</script>
