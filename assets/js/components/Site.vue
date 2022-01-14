<template>
	<div class="flex-grow-1 d-flex flex-column">
		<div class="container" @click="toggleDetails">
			<h3 class="d-none d-md-block my-4">
				{{ siteTitle || "Home" }}
			</h3>
			<Energyflow v-bind="energyflow" />
		</div>
		<div
			class="flex-grow-1 d-flex flex-column content-area"
			:style="`margin-top: ${dragTopMargin}px`"
		>
			<div class="toggle-handle py-3 d-flex justify-content-center" @click="toggleDetails">
				<shopicon-regular-arrowup
					class="toggle-icon"
					:class="`toggle-icon--${positionUp ? 'up' : 'down'}`"
				></shopicon-regular-arrowup>
			</div>
			<div class="container">
				<template v-for="(loadpoint, id) in loadpoints">
					<hr v-if="id > 0" :key="id + '_hr'" class="w-100 my-4" />
					<Loadpoint
						v-bind="loadpoint"
						:id="id"
						:key="id"
						:single="loadpoints.length === 1"
					/>
				</template>
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
			positionUp: true,
		};
	},
	computed: {
		dragTopMargin: function () {
			const min = -175;
			const max = 20;
			return this.positionUp ? min : max;
		},
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
			this.positionUp = !this.positionUp;
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
	transition: margin-top 0.4s cubic-bezier(0.5, 0.5, 0.5, 1.15);
}
.toggle-handle {
	cursor: pointer;
	color: var(--bs-gray-500);
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
