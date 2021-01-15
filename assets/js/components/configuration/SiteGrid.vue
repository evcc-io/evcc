<template>
	<CardEntry name="Netzanschluss">
		<template #status><h5>0,00 kW</h5></template>
		<template #summary>Discovergy Zähler</template>
		<template #form><Form :meters="meters" /></template>
	</CardEntry>
</template>

<script>
import CardEntry from "./CardEntry";
import Form from "./Form";

export default {
	name: "SiteGrid",
	components: { CardEntry, Form },
	props: {
		meters: {
			type: Array,
		},
	},
};
</script>
