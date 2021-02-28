<template>
	<div class="row">
		<div class="col-6 col-md-3 mt-3" v-if="gridConfigured">
			<div class="mb-2 value" v-if="gridPower > 0">
				Bezug <fa-icon icon="arrow-down" class="text-primary" />
			</div>
			<div class="mb-2 value" v-else>
				Einspeisung <fa-icon icon="arrow-up" class="text-primary"></fa-icon>
			</div>
			<h2 class="value">
				{{ fmt(gridPower) }}
				<small class="text-muted">{{ fmtUnit(gridPower) }}W</small>
			</h2>
		</div>
		<div class="col-6 col-md-3 mt-3" v-if="pvConfigured">
			<div class="mb-2 value">
				Erzeugung
				<fa-icon
					icon="sun"
					:class="{
						'text-primary': pvPower > 0,
						'text-muted': pvPower <= 0,
					}"
				></fa-icon>
			</div>
			<h2 class="value">
				{{ fmt(pvPower) }}
				<small class="text-muted">{{ fmtUnit(pvPower) }}W</small>
			</h2>
		</div>
		<div class="d-md-block col-6 col-md-3 mt-3" v-if="batteryConfigured">
			<div class="mb-2 value">
				Batterie
				<fa-icon class="text-primary" :icon="batteryIcon"></fa-icon>
			</div>
			<h2 class="value">
				{{ fmt(batteryPower) }}
				<small class="text-muted">{{ fmtUnit(batteryPower) }}W</small>
				<small class="text-muted">/</small>
				{{ batterySoC }} <small class="text-muted">%</small>
			</h2>
		</div>
	</div>
</template>

<script>
import "../icons";
import formatter from "../mixins/formatter";

const limit = 20;
const icons = [
	"battery-empty",
	"battery-quarter",
	"battery-half",
	"battery-three-quarters",
	"battery-full",
];

export default {
	name: "SiteDetails",
	props: {
		gridConfigured: Boolean,
		gridPower: Number,
		pvConfigured: Boolean,
		pvPower: Number,
		batteryConfigured: Boolean,
		batteryPower: Number,
		batterySoC: Number,
	},
	data: function () {
		return {
			iconIdx: 0,
		};
	},
	mixins: [formatter],
	computed: {
		batteryIcon: function () {
			if (Math.abs(this.batteryPower) < limit) {
				if (this.batterySoC < 30) return icons[0];
				if (this.batterySoC < 50) return icons[1];
				if (this.batterySoC < 70) return icons[2];
				if (this.batterySoC < 90) return icons[3];
				return icons[4];
			}
			return icons[this.iconIdx];
		},
	},
	mounted: function () {
		window.setInterval(() => {
			if (this.batteryPower > limit) {
				if (--this.iconIdx < 0) {
					this.iconIdx = icons.length - 1;
				}
			} else if (this.batteryPower < limit) {
				if (++this.iconIdx >= icons.length) {
					this.iconIdx = 0;
				}
			}
		}, 1000);
	},
};
</script>
