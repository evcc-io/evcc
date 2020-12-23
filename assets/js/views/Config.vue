<template>
	<div class="container">
		<!-- <h1 class="display-4 pt-3 mx-auto text-center">Konfiguration</h1>
		<p class="lead mx-auto text-center">Details der Fahrzeug-, Wallbox- und Zählerkonfiguration.</p> -->

		<div class="row mt-4 border-bottom">
			<div class="col-12">
				<p class="h1">{{ title || "Home" }}</p>
			</div>
		</div>

		<div class="row h5">
			<div class="col-md-4"></div>
			<div class="col-6 col-md-2 py-3">
				Netzzähler:
				<span class="text-primary" v-if="gridConfigured">✓</span>
				<span class="text-primary" v-else>&mdash;</span>
			</div>
			<div class="col-6 col-md-2 py-3">
				PV Zähler:
				<span class="text-primary" v-if="pvConfigured">✓</span>
				<span class="text-primary" v-else>&mdash;</span>
			</div>
			<div class="col-6 col-md-2 py-3">
				Batteriezähler:
				<span class="text-primary" v-if="batteryConfigured">✓</span>
				<span class="text-primary" v-else>&mdash;</span>
			</div>
		</div>

		<div
			v-for="(loadpoint, id) in loadpoints"
			:loadpoint="loadpoint"
			:key="id"
			:id="'loadpoint-' + id"
		>
			<div class="row mt-4 border-bottom">
				<div class="col-12">
					<p class="h1">{{ loadpoint.title || "Ladepunkt" }}</p>
				</div>
			</div>

			<div class="row h5">
				<div class="col-md-4"></div>
				<div class="col-6 col-md-2 py-3">
					Ladezähler:
					<span class="text-primary" v-if="loadpoint.chargeConfigured">✓</span>
					<span class="text-primary" v-else>&mdash;</span>
				</div>
				<div class="col-6 col-md-2 py-3">
					Phasen:
					<span class="text-primary">{{ loadpoint.phases }}p</span>
				</div>
				<div class="col-6 col-md-2 py-3">
					Min. Strom:
					<span class="text-primary">{{ loadpoint.minCurrent }}A</span>
				</div>
				<div class="col-6 col-md-2 py-3">
					Max. Strom:
					<span class="text-primary">{{ loadpoint.maxCurrent }}A</span>
				</div>
			</div>

			<div class="row h5">
				<div class="col-md-4"></div>
				<div class="col-md-8 h2">
					<div class="row py-3 h2 border-bottom">
						<div class="col-12">Fahrzeug</div>
					</div>
					<div class="row h5">
						<div class="col-6 py-3">
							Modell:
							<span class="text-primary">{{ loadpoint.socTitle || "—" }}</span>
						</div>
						<div class="col-6 py-3">
							Kapazität:
							<span class="text-primary">{{ loadpoint.socCapacity }}kWh</span>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
export default {
	name: "Config",
	data: function () {
		return this.$root.$data.store.state; // global state
	},
};
</script>
