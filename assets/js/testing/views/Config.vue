<template>
	<div class="container">
		<ul class="nav nav-tabs" id="myTab" role="tablist">
			<li class="nav-item" role="presentation">
				<a
					class="nav-link active"
					id="meter-tab"
					data-toggle="tab"
					href="#meter"
					role="tab"
					aria-controls="meter"
					aria-selected="true"
					>Meter</a
				>
			</li>
			<li class="nav-item" role="presentation">
				<a
					class="nav-link"
					id="profile-tab"
					data-toggle="tab"
					href="#charger"
					role="tab"
					aria-controls="charger"
					aria-selected="false"
					>Charger</a
				>
			</li>
			<li class="nav-item" role="presentation">
				<a
					class="nav-link"
					id="profile-tab"
					data-toggle="tab"
					href="#vehicle"
					role="tab"
					aria-controls="vehicle"
					aria-selected="false"
					>Vehicle</a
				>
			</li>
		</ul>
		<div class="tab-content" id="myTabContent">
			<div
				class="tab-pane fade show active"
				id="meter"
				role="tabpanel"
				aria-labelledby="meter-tab"
			>
				<div class="row my-3">
					<div class="col">
						<select class="form-control" v-model="meter">
							<option
								v-for="(cfg, idx) in meters"
								:key="'meter' + idx"
								:value="idx"
								:selected="idx == meter"
							>
								{{ cfg.label }}
							</option>
						</select>
					</div>
				</div>

				<Configurable
					v-bind="meters[meter]"
					:configclass="'meter'"
					:plugins="plugins"
				></Configurable>
			</div>
			<div class="tab-pane fade" id="charger" role="tabpanel" aria-labelledby="charger-tab">
				<div class="row my-3">
					<div class="col">
						<select class="form-control" v-model="charger">
							<option
								v-for="(cfg, idx) in chargers"
								:key="'charger' + idx"
								:value="idx"
								:selected="idx == charger"
							>
								{{ cfg.label }}
							</option>
						</select>
					</div>
				</div>

				<Configurable
					v-bind="chargers[charger]"
					:configclass="'charger'"
					:plugins="plugins"
				></Configurable>
			</div>
			<div class="tab-pane fade" id="vehicle" role="tabpanel" aria-labelledby="vehicle-tab">
				<div class="row my-3">
					<div class="col">
						<select class="form-control" v-model="vehicle">
							<option
								v-for="(cfg, idx) in vehicles"
								:key="'vehicle' + idx"
								:value="idx"
								:selected="idx == vehicle"
							>
								{{ cfg.label }}
							</option>
						</select>
					</div>
				</div>

				<Configurable
					v-bind="vehicles[vehicle]"
					:configclass="'vehicle'"
					:plugins="plugins"
				></Configurable>
			</div>
		</div>

		<!-- <div>
			<Ssh></Ssh>
		</div> -->
	</div>
</template>

<script>
import axios from "axios";
import Configurable from "../components/Configurable";
// import Ssh from "../components/Ssh";

export default {
	name: "Config",
	components: { Configurable },
	data: function () {
		return {
			meters: [],
			chargers: [],
			vehicles: [],
			plugins: [],
			meter: 0,
			charger: 0,
			vehicle: 0,
		};
	},
	mounted: async function () {
		try {
			this.meters = (await axios.get("/config/types/meter")).data;
			this.chargers = (await axios.get("/config/types/charger")).data;
			this.vehicles = (await axios.get("/config/types/vehicle")).data;
			this.plugins = (await axios.get("/config/types/plugin")).data;
		} catch (e) {
			window.toasts.error(e);
		}
	},
};
</script>
