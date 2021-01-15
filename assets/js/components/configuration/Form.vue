<template>
	<form>
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
		<p>
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
			<button type="button" class="btn btn-outline-secondary btn-sm" @click="edit = ''">
				abbrechen
			</button>
			&nbsp;
			<button
				type="button"
				class="btn btn-outline-primary btn-sm"
				@click.prevent="test = !test"
			>
				testen
			</button>
			&nbsp;
			<button
				type="button"
				class="btn btn-sm"
				:class="{
					'btn-outline-primary': !test,
					'btn-success': test,
				}"
				@click="edit = ''"
			>
				testen &amp; speichern
			</button>
		</p>
		<p class="text-success" v-if="test">✓ Verbindung erfolgreich hergestellt</p>
	</form>
</template>

<script>
import FormField from "./FormField";

export default {
	name: "Form",
	components: { FormField },
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
