<template>
	<CardEntry name="Erzeuger / Wechselrichter" is-configured>
		<template #status><h5 class="text-success">5,42 kW</h5></template>
		<template #summary>SMA</template>
		<template #form><Form :meters="meters" /></template>
	</CardEntry>
</template>

<script>
import CardEntry from "./CardEntry";
import Form from "./Form";

export default {
	name: "SitePv",
	components: { CardEntry, Form },
	data: function () {
		return { edit: "", extended: false, selectedMeter: "sma", test: null };
	},
	props: {
		meters: {
			type: Array,
		},
	},
	computed: {
		formFields: function () {
			const meter = this.meters.find((m) => m.type === this.selectedMeter);
			return meter ? meter.fields : [];
		},
		optionalFormFields: function () {
			return this.formFields.filter((f) => !f.required);
		},
		requiredFormFields: function () {
			return this.formFields.filter((f) => f.required);
		},
	},
};
</script>
