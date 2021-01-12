<template>
	<Card title="Hausinstallation">
		<template #content>
			<CardEntry name="PV Wechselrichter">
				<template #status><h5 class="text-success">5,42 kW</h5></template>
				<template #summary>SMA</template>
				<template #form>
					<form v-if="edit === 'pv'" class="pb-2">
						<div class="form-group">
							<label for="wechselrichter">Hersteller</label>
							<select class="custom-select" id="wechselrichter">
								<option value="e3dc">E3DC</option>
								<option value="fronius">Fronius</option>
								<option value="kostal">Kostal</option>
								<option selected value="sma">SMA</option>
								<option value="solaredge">SolarEdge</option>
								<option disabled>-----</option>
								<option value="example">Beispielanlage (5 kWp)</option>
								<option value="http">HTTP API (JSON)</option>
								<option value="modbus-tcp">Modbus-TCP</option>
							</select>
						</div>
						<div class="form-group">
							<label for="uri">Adresse (Modbus-TCP)</label>
							<input
								type="text"
								class="form-control"
								value="192.168.0.34:502"
								placeholder="192.168.0.34:502"
								id="uri"
							/>
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

export default {
	name: "Site",
	components: { Card, CardEntry },
	data: function () {
		return { edit: "", extended: false, test: null };
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
