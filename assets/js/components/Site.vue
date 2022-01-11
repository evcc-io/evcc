<template>
	<div class="flex-grow-1 d-flex flex-column" :class="{ 'user-select-none': dragActive }">
		<div class="container" @click="toggleDetails">
			<h3 class="d-none d-md-block my-4">
				{{ siteTitle || "Home" }}
			</h3>
			<Energyflow v-bind="energyflow" />
		</div>
		<div
			class="flex-grow-1 d-flex justify-content-around flex-column content-area"
			:style="`margin-top: ${dragTopMargin}px`"
			:class="{ 'content-area--transition': !dragActive }"
		>
			<div
				class="drag-handle py-3"
				@mousedown="dragStart"
				@mousemove="dragMove"
				@mouseup="dragEnd"
				@touchstart="dragStart"
				@touchmove="dragMove"
				@touchend="dragEnd"
			>
				<hr />
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
			dragStartY: 0,
			dragCurrentY: 0,
			dragActive: false,
		};
	},
	computed: {
		dragTopMargin: function () {
			const min = -175;
			const max = 20;
			let position = this.positionUp ? min : max;
			if (this.dragActive) {
				position -= this.dragStartY - this.dragCurrentY;
			}
			return Math.max(min, Math.min(position, max));
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
		dragMove(e) {
			if (this.dragActive) {
				const screenY = e.touches ? e.touches[0].screenY : e.screenY;
				this.dragCurrentY = screenY;
				e.preventDefault();
			}
		},
		dragStart(e) {
			const screenY = e.touches ? e.touches[0].screenY : e.screenY;
			this.dragActive = true;
			this.dragStartY = screenY;
			this.dragCurrentY = screenY;
			e.preventDefault();
		},
		dragEnd(e) {
			const diffY = this.dragStartY - this.dragCurrentY;
			if (diffY > 70) {
				this.positionUp = true;
			}
			if (diffY < -70) {
				this.positionUp = false;
			}
			this.dragActive = false;
			this.dragStartY = NaN;
			e.preventDefault();
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
}
.content-area--transition {
	transition: margin-top 0.4s cubic-bezier(0.5, 0.5, 0.5, 1.15);
}
.drag-handle {
	cursor: grab;
}
.drag-handle hr {
	display: block;
	margin: 0 auto;
	border: none;
	height: 2px;
	width: 1rem;
	background-color: var(--bs-gray-500);
}
</style>
