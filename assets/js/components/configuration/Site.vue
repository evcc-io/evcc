<template>
	<Card title="Hausinstallation">
		<template #content>
			<CardEntry name="PV Wechselrichter">
				<template #status><h5 class="text-success">5,42 kW</h5></template>
				<template #summary>SMA</template>
				<template #form>
					<form>
						<div class="form-group">
							<label for="wechselrichter">Hersteller</label>
							<select
								class="custom-select"
								id="wechselrichter"
								v-model="selectedMeter"
							>
								<option
									:value="meter.type"
									:key="meter.type"
									v-for="meter in meters"
								>
									{{ meter.label }}
								</option>
								<!--<option value="e3dc">E3DC</option>
								<option value="fronius">Fronius</option>
								<option value="kostal">Kostal</option>
								<option selected value="sma">SMA</option>
								<option value="solaredge">SolarEdge</option>
								<option disabled>-----</option>
								<option value="example">Beispielanlage (5 kWp)</option>
								<option value="http">HTTP API (JSON)</option>
								<option value="modbus-tcp">Modbus-TCP</option>-->
							</select>
						</div>
						<div
							class="form-group"
							:key="formField.name"
							v-for="formField in formFields"
						>
							<label :for="formField.name">
								{{ formField.label }}
								<small class="text-muted" v-if="!formField.required">
									(optional)
								</small>
							</label>
							<input
								type="text"
								v-if="formField.type === 'string' && !formField.enum"
								class="form-control"
								:placeholder="formField.default"
								value=""
								:id="formField.name"
							/>
							<input
								type="number"
								v-if="formField.type === 'int' || formField.type === 'uint8'"
								class="form-control"
								style="width: 50%"
								:placeholder="formField.default"
								value=""
								:id="formField.name"
							/>
							<select
								v-if="formField.type === 'string' && formField.enum"
								class="custom-select"
								:id="formField.name"
							>
								<option v-if="!formField.required" value="">
									- bitte wählen -
								</option>
								<option :key="value" :value="value" v-for="value in formField.enum">
									{{ value }}
								</option>
							</select>
						</div>
						<p>
							<a href="#" @click.prevent="extended = !extended">
								erweiterte Einstellungen
								<span v-if="!extended">anzeigen</span>
								<span v-else>ausblenden</span>
							</a>
						</p>
						<div v-if="extended">
							<div class="form-group">
								<label for="power">
									Modbus Slave ID
									<small
										title="Der Standardwert bei SMA Wechselrichtern ist 126. Lorem ipsum ..."
										class="text-muted"
									>
										Was ist das?
									</small>
								</label>
								<input
									type="text"
									class="form-control"
									placeholder="126"
									id="power"
								/>
							</div>
							<div class="form-group">
								<label for="power">
									Power
									<small>(optional)</small>
								</label>
								<input
									type="text"
									class="form-control"
									placeholder="Power"
									id="power"
								/>
							</div>
							<div class="form-group">
								<label for="energy">
									Zählerstand
									<small>(optional)</small>
								</label>
								<input
									type="text"
									class="form-control"
									placeholder="Sum"
									id="energy"
								/>
							</div>
						</div>
						<p>
							<button
								type="button"
								class="btn btn-outline-secondary btn-sm"
								@click="edit = ''"
							>
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
			</CardEntry>
			<CardEntry name="Hausverbrauch">
				<template #status><h5>4,20 kW</h5></template>
				<template #summary>abgeleitet</template>
				<template #form><input placeholder="hallo welt!" /></template>
			</CardEntry>
			<CardEntry name="Hausbatterie">
				<template #status>
					<h5 class="text-success mb-0">4,20 kW</h5>
					<small class="text-muted">76%</small>
				</template>
				<template #summary>BYD B-BOX PREMIUM 9.0kWh</template>
				<template #form><input placeholder="hallo welt!" /></template>
			</CardEntry>
			<CardEntry name="Netzanschluss">
				<template #status><h5>0,00 kW</h5></template>
				<template #summary>Discovergy Zähler</template>
				<template #form><input placeholder="hallo welt!" /></template>
			</CardEntry>
		</template>
	</Card>
</template>

<script>
import Card from "./Card";
import CardEntry from "./CardEntry";
import meters from "./meter.json";

export default {
	name: "Site",
	components: { Card, CardEntry },
	data: function () {
		return { edit: "", extended: false, selectedMeter: "sma", test: null };
	},
	props: {
		meters: {
			type: Object,
			default: meters,
		},
	},
	computed: {
		formFields: function () {
			const meter = this.meters.find((m) => m.type === this.selectedMeter);
			return meter.fields;
		},
	},
};
</script>

<style scoped>
.card-header-with-link {
	display: flex;
	justify-content: space-between;
	align-items: baseline;
}
</style>
