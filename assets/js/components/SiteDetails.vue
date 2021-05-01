<template>
	<div class="row justify-content-md-start overflow-hidden" :class="`row-cols-${numberOfPanels}`">
		<div class="px-3" :class="panelClass" v-if="gridConfigured">
			<div class="mb-2 value" v-if="gridPower > 0">
				Bezug <fa-icon icon="arrow-down" class="text-primary" />
			</div>
			<div class="mb-2 value" v-else>
				Einspeisung <fa-icon icon="arrow-up" class="text-primary"></fa-icon>
			</div>
			<h3 class="value">
				{{ fmt(gridPower) }}
				<small class="text-muted">{{ fmtUnit(gridPower) }}W</small>
			</h3>
		</div>

		<div class="px-3" :class="panelClass" v-if="pvConfigured">
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
			<h3 class="value">
				{{ fmt(pvPower) }}
				<small class="text-muted">{{ fmtUnit(pvPower) }}W</small>
			</h3>
		</div>
		<div class="px-3" :class="panelClass" v-if="batteryConfigured">
			<div class="mb-2 value">
				<div class="d-block d-sm-none">
					Akku <span class="text-muted"> / {{ batterySoC }} %</span>
				</div>
				<div class="d-none d-sm-block">
					Batterie <span class="text-muted"> / {{ batterySoC }}% </span>
					<fa-icon class="text-primary" :icon="batteryIcon"></fa-icon>
				</div>
			</div>
			<h3 class="value">
				{{ fmt(batteryPower) }}
				<small class="text-muted">{{ fmtUnit(batteryPower) }}W</small>
			</h3>
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
		numberOfPanels: function () {
			let count = 0;
			if (this.gridConfigured) count++;
			if (this.pvConfigured) count++;
			if (this.batteryConfigured) count++;
			return count;
		},
		panelClass: function () {
			if (this.numberOfPanels == 3) {
				return "col-4 col-md-4 col-lg-3";
			}
			if (this.numberOfPanels == 2) {
				return "col-6 col-md-4 col-lg-6";
			}
			return "col-12";
		},
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
