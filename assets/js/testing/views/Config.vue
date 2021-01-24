<template>
	<div class="container">
		<h3 class="my-4">Class: meters</h3>

		<div class="row">
			<div class="col">
				<select class="form-control" v-model="meter">
					<option
						v-for="(cfg, idx) in meters"
						:key="idx"
						:value="idx"
						:selected="idx == meter"
					>
						{{ cfg.label }}
					</option>
				</select>
			</div>
		</div>

		<Element v-bind="meters[meter]" :configclass="'meter'" :plugins="plugins"></Element>

		<h3 class="my-4">Class: chargers</h3>

		<div class="row">
			<div class="col">
				<select class="form-control" v-model="charger">
					<option
						v-for="(cfg, idx) in chargers"
						:key="idx"
						:value="idx"
						:selected="idx == charger"
					>
						{{ cfg.label }}
					</option>
				</select>
			</div>
		</div>

		<Element v-bind="chargers[charger]" :configclass="'charger'" :plugins="plugins"></Element>

		<!-- <div>
			<Ssh></Ssh>
		</div> -->
	</div>
</template>

<script>
import axios from "axios";
import Element from "../components/Element";
import Ssh from "../components/Ssh";

export default {
	name: "Config",
	components: { Element, Ssh },
	data: function () {
		return {
			meters: [],
			chargers: [],
			plugins: [],
			meter: 0,
			charger: 0,
		};
	},
	mounted: async function () {
		try {
			this.meters = (await axios.get("/config/types/meter")).data;
			this.chargers = (await axios.get("/config/types/charger")).data;
			this.plugins = (await axios.get("/config/types/plugin")).data;
			console.log(this.meters);
			console.log(this.plugins);
		} catch (e) {
			window.toasts.error(e);
		}
	},
};
</script>
