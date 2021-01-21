<template>
	<CardEntry name="Hausbatterie" is-configured>
		<template #status>
			<h5 class="text-success mb-0">4,20 kW</h5>
			<small class="text-muted">76%</small>
		</template>
		<template #summary>BYD B-BOX PREMIUM 9.0kWh</template>
		<template #form><Form :meters="meters" /></template>
	</CardEntry>
</template>

<script>
import CardEntry from "./CardEntry";
import Form from "./Form";

export default {
	name: "SiteBattery",
	components: { CardEntry, Form },

	props: {
		meters: {
			type: Array,
		},
	},
};
</script>
