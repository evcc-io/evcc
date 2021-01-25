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

		<Configurable v-bind="types[type]" :configclass="klass" :plugins="plugins"></Configurable>
	</div>
</template>

<script>
import axios from "axios";
import Configurable from "../components/Configurable";

export default {
	name: "ConfigClass",
	components: { Configurable },
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
	mounted: async function () {
		try {
			this.types = (await axios.get("/config/types/" + this.klass)).data;
		} catch (e) {
			window.toasts.error(e);
		}
	},
};
</script>
