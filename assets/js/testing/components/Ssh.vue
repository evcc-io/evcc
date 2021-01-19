<template>
	<div class="container">
		<h3>Fernwartung</h3>

		SSH:
		<span v-if="!updated">unbekannt</span>
		<button type="button" class="btn btn-primary" @click="setActive(false)" v-else-if="active">
			Stop
		</button>
		<button type="button" class="btn btn-primary" @click="setActive(true)" v-else>Start</button>
	</div>
</template>

<script>
import axios from "axios";

export default {
	name: "Ssh",
	data: function () {
		return {
			updated: false,
			active: false,
		};
	},
	methods: {
		setActive: function (active) {
			axios
				.post("/ssh", '{"active":' + active.toString() + "}")
				.then(() => {
					this.active = active;
				})
				.catch(window.toasts.error);
		},
	},
	mounted: function () {
		axios
			.get("/ssh")
			.then((resp) => {
				console.log(resp.data);
				this.updated = true;
				this.active = resp.data.active;
			})
			.catch(window.toasts.error);
	},
};
</script>
