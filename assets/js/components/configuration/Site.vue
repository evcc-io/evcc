<template>
	<Card title="Hausinstallation">
		<template #content>
			<SiteGrid :meters="metersFor('grid')" />
			<SitePv :meters="metersFor('pv')" />
			<SiteBattery :meters="metersFor('battery')" />
		</template>
	</Card>
</template>

<script>
import Card from "./Card";
import meters from "./meter.json";
import SiteGrid from "./SiteGrid";
import SitePv from "./SitePv";
import SiteBattery from "./SiteBattery";

export default {
	name: "Site",
	components: { Card, SiteGrid, SitePv, SiteBattery },
	props: {
		meters: {
			type: Array,
			default: () => meters,
		},
	},
	methods: {
		metersFor: function () {
			// TODO: filter meters to only show appropriate ones
			return this.meters;
		},
	},
};
</script>
