<template>
	<div class="container">
		<div class="row mt-4">
			<div class="col-12">
				<h1>Setup</h1>
			</div>
		</div>
		<div class="card-deck mt-4 mb-5">
			<div class="card">
				<div class="card-header card-header-with-link">
					<h3 class="mb-0">Haus</h3>
					<a class="text" href="#">umbenennen</a>
				</div>
				<div class="card-body">
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">PV Wechselrichter</h5>
						&nbsp;
						<a href="#" @click.prevent="edit = 'pv'" v-show="edit !== 'pv'">ändern</a>
						<h5 class="float-right text-success">5,42 kW</h5>
					</div>
					<transition name="fade" mode="out-in">
						<form v-if="edit === 'pv'">
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
							<p class="text-success" v-if="test">
								✓ Verbindung erfolgreich hergestellt
							</p>
							<p class="pb-2">
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
						</form>
						<p class="card-text" v-else>SMA</p>
					</transition>
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Hausverbrauch</h5>
						&nbsp;
						<a href="#" @click.prevent="edit = 'house'">ändern</a>
						<h5 class="float-right">4,20 kW</h5>
					</div>
					<p class="card-text">abgeleitet</p>
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Hausbatterie</h5>
						&nbsp;
						<a href="#">ändern</a>
						<span class="float-right text-right">
							<h5 class="text-success mb-0">4,20 kW</h5>
							<small class="text-muted">76%</small>
						</span>
					</div>
					<p class="card-text">BYD B-BOX PREMIUM 9.0kWh</p>
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Netzanschluss</h5>
						&nbsp;
						<a href="#">ändern</a>
						<h5 class="float-right">0,00 kW</h5>
					</div>
					<p class="card-text">Discovergy Zähler</p>
				</div>
			</div>
			<div class="card">
				<div class="card-header card-header-with-link">
					<h3 class="mb-0">Fahrzeuge</h3>
					<a class="text" href="#">hinzufügen</a>
				</div>
				<div class="card-body">
					<h5 class="card-title">
						VW ID.3
						<span class="float-right text-right">
							76 %<br />
							<small class="text-muted">lädt 3,7 kW</small>
						</span>
					</h5>
					<p class="card-text">Volkswagen Connect <a href="#">ändern</a></p>
					<h5 class="card-title">
						Tesla Model 3
						<span class="float-right text-right">
							33 %<br />
							<small class="text-muted">schläft</small>
						</span>
					</h5>
					<p class="card-text">Tesla API <a href="#">ändern</a></p>
				</div>
			</div>
		</div>
		<div class="my-4">
			<h2 style="display: inline-block" class="pr-3">Ladepunkte</h2>
			<a href="#">hinzufügen</a>
		</div>

		<div class="card-deck mt-4 mb-5">
			<div class="card">
				<div class="card-header card-header-with-link">
					<h4 class="mb-0">Carport</h4>
					<a class="text" href="#">umbenennen</a>
				</div>
				<div class="card-body">
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Wallbox</h5>
						&nbsp;
						<a href="#">ändern</a>
						<span class="float-right text-right">
							<h5 class="mb-0">3,7 kW</h5>
							<small class="text-muted">lädt</small>
						</span>
					</div>
					<p class="card-text">NRGKick Connect</p>
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Fahrzeuge</h5>
						&nbsp;
						<a href="#">ändern</a>
					</div>
					<p class="card-text">VW ID.3</p>
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Ladeverhalten</h5>
						&nbsp;
						<a href="#">ändern</a>
					</div>
					<p class="card-text">
						Modus: PV-only<br />
						Ladeziel: 90% (sofort 20%)<br />
						Ladeleistung: 6A bis 16A, 3-phasig<br />
					</p>
				</div>
				<div class="card-footer text-right">
					<a class="text-danger" href="#">entfernen</a>
				</div>
			</div>
			<div class="card">
				<div class="card-header card-header-with-link">
					<h4 class="mb-0">Garage</h4>
					<a class="text" href="#">umbenennen</a>
				</div>
				<div class="card-body">
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Wallbox</h5>
						&nbsp;
						<a href="#">ändern</a>
						<span class="float-right text-right">
							<h5 class="mb-0">0 kW</h5>
							<small class="text-muted">getrennt</small>
						</span>
					</div>
					<p class="card-text">KEBA X30</p>
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Fahrzeuge</h5>
						&nbsp;
						<a href="#">ändern</a>
					</div>
					<p class="card-text">
						Tesla Model 3<br />
						VW ID.3
					</p>
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Ladeverhalten</h5>
						&nbsp;
						<a href="#">ändern</a>
					</div>
					<p class="card-text">
						Modus: schnell<br />
						Ladeziel: 90%<br />
						Ladeleistung: 6A bis 32A, 3-phasig<br />
					</p>
				</div>
				<div class="card-footer text-right">
					<a class="text-danger" href="#">entfernen</a>
				</div>
			</div>
		</div>
		<h2 class="my-4">Weitere Einstellungen</h2>
		<div class="card-deck">
			<div class="card">
				<div class="card-header card-header-with-link">
					<h4 class="mb-0">Schnittstellen</h4>
				</div>
				<div class="card-body">
					<h5 class="card-title">MQTT</h5>
					<p class="card-text">nicht konfiguriert <a href="#">ändern</a></p>
					<h5 class="card-title">
						InfluxDB
						<span class="float-right text-right">
							<small class="text-success">verbunden</small>
						</span>
					</h5>
					<p class="card-text">https://influx.local/ <a href="#">ändern</a></p>
				</div>
			</div>
			<div class="card">
				<div class="card-header card-header-with-link">
					<h4 class="mb-0">Benachrichtigung</h4>
				</div>
				<div class="card-body">
					<h5 class="card-title">Pushover</h5>
					<p class="card-text">nicht konfiguriert <a href="#">ändern</a></p>
					<h5 class="card-title">Telegram</h5>
					<p class="card-text">nicht konfiguriert <a href="#">ändern</a></p>
					<h5 class="card-title">E-Mail</h5>
					<p class="card-text">nicht konfiguriert <a href="#">ändern</a></p>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
export default {
	name: "Setup",
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
.fade-leave {
	opacity: 1;
}
.fade-leave-active {
	transition: opacity 0.2s;
}
.fade-leave-to {
	opacity: 0;
}
.fade-enter {
	opacity: 0;
}
.fade-enter-active {
	transition: opacity 0.2s;
}
.fade-enter-to {
	opacity: 1;
}
form {
	width: 75%;
}
</style>
