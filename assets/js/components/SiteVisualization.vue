<template>
	<div>
		<div class="labels d-flex justify-content-between">
			<div>
				Verbrauch:
				<strong class="label-usage">{{ kw(usage) }}</strong>
			</div>
			<div>
				Einspeisung:
				<strong class="label-pv-export">{{ kw(pvExport) }}</strong>
			</div>
		</div>
		<div class="site-progress">
			<div class="usage" :style="{ width: width(usage) }"></div>
			<!--
			<div class="grid-usage" :style="{ width: width(gridUsage) }">
				<span class="label">Bezug</span>
				<span class="detail">{{ kw(gridUsage) }}</span>
			</div>
			<div class="pv-usage" :style="{ width: width(pvUsage) }">
				<span class="label">Eigenverbrauch</span>
				<span class="detail">{{ kw(pvUsage) }}</span>
			</div>
			-->
			<div class="pv-export" :style="{ width: width(pvExport) }"></div>
		</div>
		<div class="site-charger">
			<div
				class="charger1"
				:style="{
					width: width(Math.min(usage, loadpoints[0].chargePower)),
					marginRight: width(pvExport),
				}"
			>
				<span>{{ kw(Math.min(usage, loadpoints[0].chargePower)) }}</span>
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
		usage: function () {
			return Math.max(0, this.pvPower + this.gridPower);
		},
		gridUsage: function () {
			return Math.max(0, this.gridPower);
		},
		pvUsage: function () {
			return Math.min(this.pvPower, this.pvPower + this.gridPower);
		},
		pvExport: function () {
			return this.pvPower - this.pvUsage;
		},
		max: function () {
			return this.usage + this.pvExport;
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
	margin: 1rem 0;
	height: 1.5rem;
	display: flex;
}
.site-progress .label {
	display: none;
}
.site-progress .detail {
	display: block;
}

.usage {
	background-color: #999;
	color: #eee;
	border-radius: 5px;
}
.grid-usage {
	background-color: #18191a;
	color: #eee;
}
.pv-usage {
	background-color: #66d85a;
	color: #18191a;
}
.pv-export {
	background-color: #fbdf4b;
	color: #18191a;
	margin-left: 10px;
	border-radius: 5px;
}
.usage,
.grid-usage,
.pv-usage,
.pv-export {
	overflow-x: hidden;
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
