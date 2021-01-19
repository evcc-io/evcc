<template>
	<div class="container">
		<div>
			<Ssh></Ssh>
		</div>

		<h3>Class: meters</h3>
		<Element
			v-for="(cfg, idx) in meters"
			v-bind="cfg"
			:key="idx"
			:configclass="'meters'"
		></Element>

		<h3>Class: chargers</h3>
		<Element
			v-for="(cfg, idx) in chargers"
			v-bind="cfg"
			:key="idx"
			:configclass="'chargers'"
		></Element>
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
