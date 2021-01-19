<template>
	<form @submit.prevent="testAndSave">
		<div class="form-group">
			<label for="wechselrichter">Messgerät</label>
			<select class="custom-select" id="wechselrichter" v-model="selectedMeter">
				<option :value="meter.type" :key="meter.type" v-for="meter in meters">
					{{ meter.label }}
				</option>
			</select>
		</div>
		<FormField
			v-bind="formField"
			:key="formField.name"
			v-for="formField in requiredFormFields"
		/>
		<p v-if="optionalFormFields.length > 0">
			<a href="#" @click.prevent="extended = !extended">
				erweiterte Einstellungen
				<span v-if="!extended">anzeigen</span>
				<span v-else>ausblenden</span>
			</a>
		</p>
		<div v-if="extended">
			<FormField
				v-bind="formField"
				:key="formField.name"
				v-for="formField in optionalFormFields"
			/>
		</div>
		<p>
			<button
				type="button"
				class="btn btn-outline-secondary btn-sm"
				@click="() => $emit('close')"
			>
				abbrechen
			</button>
			&nbsp;
			<button type="button" class="btn btn-outline-primary btn-sm" @click.prevent="test">
				testen
			</button>
			&nbsp;
			<button
				type="submit"
				class="btn btn-sm"
				:class="{
					'btn-outline-primary': !tested,
					'btn-success': tested,
				}"
				@click="edit = ''"
			>
				testen &amp; speichern
			</button>
		</p>
		<p class="text-success" v-if="tested">✓ Verbindung erfolgreich hergestellt</p>
	</form>
</template>

<script>
import FormField from "./FormField";
import axios from "axios";

export default {
	name: "Form",
	components: { FormField },
	data: function () {
		return { edit: "", extended: false, selectedMeter: "sma", tested: false };
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
	methods: {
		test: async function () {
			this.tested = true;
		},
		testAndSave: async function (e) {
			const formData = new FormData(e.target);
			console.log({ e, formData });
			try {
				await axios.post("/api/config/meter/grid", formData);
				this.tested = false;
			} catch (e) {
				console.error(e);
			}
		},
	},
};
</script>
