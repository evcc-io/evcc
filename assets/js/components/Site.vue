<template>
	<div class="flex-grow-1 d-flex flex-column">
		<div class="container" @click="toggleDetails">
			<h2 class="d-block my-4">
				{{ siteTitle || "Home" }}
			</h2>
			<Energyflow v-bind="energyflow" :details-visible="detailsVisible" />
		</div>
		<div class="flex-grow-1 d-flex flex-column content-area">
			<div class="toggle-handle py-3 d-flex justify-content-center" @click="toggleDetails">
				<shopicon-regular-arrowup
					class="toggle-icon"
					:class="`toggle-icon--${detailsVisible ? 'up' : 'down'}`"
				></shopicon-regular-arrowup>
			</div>
			<div class="container">
				<h2 class="mb-3 mb-sm-4">Ladepunkte</h2>
			</div>
			<div class="container px-0">
				<div class="d-block d-xl-flex flex-wrap">
					<div
						v-for="(loadpoint, id) in loadpoints"
						:key="id"
						class="flex-grow-1 me-xl-4 mx-xl-2 pb-2"
					>
						<Loadpoint v-bind="loadpoint" :id="id" :single="loadpoints.length === 1" />
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/arrowup";
import Energyflow from "./Energyflow";
import Loadpoint from "./Loadpoint";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";

export default {
	name: "Site",
	components: { Loadpoint, Energyflow },
	mixins: [formatter, collector],
	props: {
		loadpoints: Array,

		// details
		gridConfigured: Boolean,
		gridPower: Number,
		homePower: Number,
		pvConfigured: Boolean,
		pvPower: Number,
		batteryConfigured: Boolean,
		batteryPower: Number,
		batterySoC: Number,
		gridCurrents: Array,
		prioritySoC: Number,
		siteTitle: String,
	},
	data: function () {
		return {
			detailsVisible: false,
			upperHeight: 0,
		};
	},
	computed: {
		energyflow: function () {
			return this.collectProps(Energyflow);
		},
		activeLoadpointsCount: function () {
			return this.loadpoints.filter((lp) => lp.chargePower > 0).length;
		},
		loadpointsPower: function () {
			return this.loadpoints.reduce((sum, lp) => {
				sum += lp.chargePower || 0;
				return sum;
			}, 0);
		},
	},
	methods: {
		toggleDetails() {
			this.detailsVisible = !this.detailsVisible;
		},
	},
};
</script>
<style scoped>
.content-area {
	background-color: var(--bs-gray-dark);
	border-radius: 20px 20px 0 0;
	color: var(--bs-white);
	z-index: 10;
	min-height: 90vh;
}
.toggle-handle {
	cursor: pointer;
	color: var(--bs-gray-medium);
}

.toggle-icon {
	transition: transform 0.3s linear;
}
.toggle-icon--up {
	transform: scaleY(-1);
}
.toggle-icon--down {
	transform: scaleY(1);
}
</style>
