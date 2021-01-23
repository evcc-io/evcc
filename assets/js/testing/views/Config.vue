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

		<Element v-bind="meters[meter]" :configclass="'meters'"></Element>

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

		<Element v-bind="chargers[charger]" :configclass="'chargers'"></Element>

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
			meters: {},
			chargers: {},
			meter: 0,
			charger: 0,
		};
	},
	mounted: function () {
		axios
			.get("/config/types/meter")
			.then((resp) => {
				this.meters = resp.data;
			})
			.catch(window.toasts.error);
		axios
			.get("/config/types/charger")
			.then((resp) => {
				this.chargers = resp.data;
			})
			.catch(window.toasts.error);
	},
};
</script>
