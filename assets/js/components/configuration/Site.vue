<template>
	<Card title="Hausinstallation">
		<template #content>
			<CardEntry
				name="Netzanschluss"
				is-required
				:edit-mode="editMode === 'grid'"
				@open="open('grid')"
				@close="close"
			>
				<template #status><h5>0,00 kW</h5></template>
				<template #summary></template>
				<template #form><Form :meters="metersFor('grid')" usage="grid" /></template>
			</CardEntry>
			<CardEntry
				name="Erzeuger / Wechselrichter"
				is-configured
				:edit-mode="editMode === 'pv'"
				@open="open('pv')"
				@close="close"
			>
				<template #status><h5 class="text-success">5,42 kW</h5></template>
				<template #summary>SMA</template>
				<template #form><Form :meters="metersFor('pv')" usage="pv" /></template>
			</CardEntry>
			<CardEntry
				name="Hausbatterie"
				is-configured
				:edit-mode="editMode === 'battery'"
				@open="open('battery')"
				@close="close"
			>
				<template #status>
					<h5 class="text-success mb-0">4,20 kW</h5>
					<small class="text-muted">76%</small>
				</template>
				<template #summary>BYD B-BOX PREMIUM 9.0kWh</template>
				<template #form><Form :meters="metersFor('battery')" usage="battery" /></template>
			</CardEntry>
		</template>
	</Card>
</template>

<script>
import Card from "./Card";
import CardEntry from "./CardEntry";
import Form from "./Form";

export default {
	name: "Site",
	components: { Card, CardEntry, Form },
	props: {
		meters: {
			type: Array,
		},
	},
	data: function () {
		return { editMode: null };
	},
	methods: {
		metersFor: function () {
			// TODO: filter meters to only show appropriate ones
			return this.meters;
		},
		open: function (name) {
			this.editMode = name;
		},
		close: function () {
			this.editMode = null;
		},
	},
};
</script>
