<template>
	<div class="container">
		<div class="row mt-4">
			<div class="col-12">
				<h1>Setup</h1>
			</div>
		</div>
		<div class="card-deck mt-4 mb-5">
			<Site :meters="meters" />
			<Vehicles :vehicles="vehicles" />
		</div>
		<Loadpoints :chargers="chargers" />

		<h2 class="my-4">Weitere Einstellungen</h2>
		<div class="card-deck">
			<Interfaces />
			<Notifications />
		</div>
	</div>
</template>

<script>
import axios from "axios";

import Site from "../components/configuration/Site";
import Vehicles from "../components/configuration/Vehicles";
import Loadpoints from "../components/configuration/Loadpoints";
import Interfaces from "../components/configuration/Interfaces";
import Notifications from "../components/configuration/Notifications";

export default {
	name: "Setup",
	components: { Site, Vehicles, Loadpoints, Interfaces, Notifications },
	data: function () {
		return {
			meters: [],
			vehicles: [],
			chargers: [],
		};
	},
	mounted: async function () {
		try {
			this.meters = (await axios.get("/config/types/meter")).data;
			this.vehicles = (await axios.get("/config/types/vehicle")).data;
			this.chargers = (await axios.get("/config/types/charger")).data;
		} catch (e) {
			window.toasts.error(e);
		}
	},
};
</script>
