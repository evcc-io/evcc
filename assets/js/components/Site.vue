<template>
	<div class="flex-grow-1 d-flex flex-column">
		<div ref="upper" class="container" @click="toggleDetails">
			<h2 class="d-block my-4">
				{{ siteTitle || "Home" }}
			</h2>
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
			<div class="container px-0">
				<h2 class="mb-3 mb-sm-4 px-2 mx-1">Ladepunkte</h2>
				<template v-for="(loadpoint, id) in loadpoints">
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
			positionUp: false,
			upperHeight: 0,
		};
	},
	computed: {
		dragTopMargin: function () {
			const visualizationHeight = 175;
			const min = -1 * this.upperHeight + visualizationHeight;
			const max = 0;
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
	mounted() {
		this.updateUpperHeight();
		window.addEventListener("resize", this.updateUpperHeight);
	},
	destroyed() {
		window.removeEventListener("resize", this.updateUpperHeight);
	},
	methods: {
		updateUpperHeight() {
			this.upperHeight = this.$refs.upper.offsetHeight;
		},
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
