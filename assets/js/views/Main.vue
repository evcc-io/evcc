<template>
	<div class="container">
		<Site v-bind:state="state" v-if="configured"></Site>
		<div v-else>
			<div class="row py-5">
				<div class="col12">
					<p class="h1 pt-5 pb-2 border-bottom">Willkommen bei evcc</p>
					<p class="lead pt-2">
						<b>evcc</b> ist dient zur flexiblen Ladesteuerung von Elektrofahrzeugen.
					</p>
					<p class="pt-2">
						Es sieht aus, als wäre Dein <b>evcc</b> noch nicht konfiguriert. Um
						<b>evcc</b> zu konfigurieren sind die folgenden Schritte notwendig:
					</p>
					<ol class="pt-2">
						<li>
							Erzeugen einer Konfigurationsdatei mit Namen
							<code>evcc.yaml</code>. Die Standardkonfiguration
							<code>evcc.dist.yaml</code> kann dafür als Vorlage dienen (<a
								href="https://github.com/andig/evcc/blob/master/evcc.dist.yaml"
								>Download</a
							>).
						</li>
						<li>Konfiguration der Wallbox als <code>chargers</code>.</li>
						<li>
							Konfiguration des EVU Zählers und evtl. weiterer Zähler unter
							<code>meters</code>.
						</li>
						<li>
							Konfiguration des Netzanschlusses unter
							<code>site</code>. In einer Site wird der Netzanschluss mit dem
							konfigurierten EVU Zähler (<code>meter</code>) verbunden.
						</li>
						<li>
							Konfiguration eines Ladepunktes unter
							<code>loadpoints</code>. In einem Ladepunkt wird die konfigurierte
							Wallbox (<code>charger</code>) mit dem Ladepunkt verbunden.
						</li>
						<li>
							Start von <b>evcc</b> mit der neu erstellten Konfiguration:
							<code>evcc -c evcc.yaml</code>
						</li>
					</ol>
					<p>Minimale Beispielkonfiguration für <b>evcc</b>:</p>
					<p>
						<code>
							<pre class="mx-3">
                uri: localhost:7070 # Adresse für UI
                interval: 10s # Regelintervall
                meters:
                - name: evu-zähler
                type: ... # Detailkonfiguration des EVU Zählers
                - name: ladezähler
                type: ... # Detailkonfiguration des Ladezählers (optional)
                chargers:
                - name: wallbox
                type: ... # Detailkonfiguration der Wallbox
                site:
                  title: Home
                  meters:
                  grid: evu-zähler # EVU Zähler
                loadpoints:
                - title: Ladepunkt # ui display name
                  charger: wallbox # charger
                  meters:
                    charge: ladezähler # Ladezählers (optional)
              </pre
							>
						</code>
					</p>
					<p>
						Viel Spass mit <b>evcc</b>! Bei Problemen kannst Du uns auf
						<a href="https://github.com/andig/evcc/issues">GitHub</a>
						erreichen.
					</p>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import Site from "../components/Site";

export default {
	name: "Main",
	components: { Site },
	data: function () {
		return {
			state: this.$root.$data.store.state, // global state
		};
	},
	computed: {
		configured: function () {
			const val = window.evcc.configured;
			// for development purposes
			if (val == window.evcc.configured) {
				return true;
			}
			if (!isNaN(parseInt(val)) && parseInt(val) > 0) {
				return true;
			}
			return false;
		},
	},
};
</script>
