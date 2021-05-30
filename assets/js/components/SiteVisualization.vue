<template>
	<div>
		<div class="site-progress">
			<div class="grid-used" :style="{ width: width(gridUsed) }">
				<span class="label">Bezug</span>
				<span class="detail">{{ kw(gridUsed) }}</span>
			</div>
			<div class="pv-used" :style="{ width: width(pvUsed) }">
				<span class="label">Eigenverbrauch</span>
				<span class="detail">{{ kw(pvUsed) }}</span>
			</div>
			<div class="pv-available" :style="{ width: width(pvAvailable) }">
				<span class="label">Einspeisung</span>
				<span class="detail">{{ kw(pvAvailable) }}</span>
			</div>
		</div>
		<div class="site-charger">
			<div
				class="charger1"
				:style="{
					width: width(Math.min(used, loadpoints[0].chargePower)),
					marginRight: width(pvAvailable),
				}"
			>
				<span>{{ kw(Math.min(used, loadpoints[0].chargePower)) }}</span>
			</div>
		</div>
	</div>
</template>

<script>
import "../icons";
import formatter from "../mixins/formatter";

export default {
	name: "SiteVisualization",
	props: {
		gridConfigured: Boolean,
		gridPower: Number,
		pvConfigured: Boolean,
		pvPower: Number,
		batteryConfigured: Boolean,
		batteryPower: Number,
		batterySoC: Number,
		loadpoints: Array,
	},
	mixins: [formatter],
	computed: {
		used: function () {
			return this.pvPower + this.gridPower;
		},
		gridUsed: function () {
			return Math.max(0, this.gridPower);
		},
		pvUsed: function () {
			return Math.min(this.pvPower, this.pvPower + this.gridPower);
		},
		pvAvailable: function () {
			return this.pvPower - this.pvUsed;
		},
		max: function () {
			return this.gridUsed + this.pvUsed + this.pvAvailable;
		},
	},
	methods: {
		width: function (power) {
			return (100 / this.max) * power + "%";
		},
		kw: function (watt) {
			return (watt / 1000).toFixed(1) + " kW";
		},
	},
};
</script>
<style scoped>
.site-progress {
	margin: 3rem 0 1rem;
	height: 1.5rem;
	background-color: #eee;
	display: flex;
}
.site-progress .label {
	display: none;
}
.site-progress .detail {
	display: block;
}

.grid-used {
	background-color: #18191a;
	color: #eee;
}
.pv-used {
	background-color: #66d85a;
	color: #18191a;
}
.pv-available {
	background-color: #fbdf4b;
	color: #18191a;
}
.grid-used,
.pv-used,
.pv-available {
	overflow-x: hidden;
	border-right: 1px solid #eee;
	display: flex;
	justify-content: center;
	align-items: center;
}

.site-charger {
	margin: 1rem 0;
	padding: 1rem 0;
	display: flex;
	justify-content: flex-end;
}
.charger1 {
	position: relative;
	background-color: #18191a;
	text-align: center;
	height: 1px;
}
.charger1::before,
.charger1::after {
	content: "";
	border-left: 1px solid #18191a;
	position: absolute;
	top: -5px;
	height: 11px;
}
.charger1::before {
	left: 0;
}
.charger1::after {
	right: 0;
}
.charger1 span {
	background-color: #fff;
	top: -0.7em;
	line-height: 1em;
	position: relative;
	padding: 0.5em;
}
</style>
