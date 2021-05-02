<template>
	<div>
		<button
			href="#"
			data-bs-toggle="modal"
			data-bs-target="#updateModal"
			v-if="newVersionAvailable"
			class="btn btn-link ps-0 text-decoration-none link-dark text-nowrap"
		>
			<fa-icon icon="gift" class="icon me-1"></fa-icon>
			Update<span class="d-none d-sm-inline"> verfügbar</span>: {{ available }}
		</button>
		<a
			:href="releaseNotesUrl(installed)"
			target="_blank"
			class="btn btn-link ps-0 text-decoration-none link-dark text-nowrap"
			v-else
		>
			Version {{ installed }}
		</a>

		<div id="updateModal" class="modal fade" tabindex="-1" role="dialog" aria-hidden="true">
			<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">Update verfügbar</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<div v-if="updateStarted">
							<p>Nach der Aktualisierung wird evcc neu gestartet.</p>
							<div class="progress my-3">
								<div
									class="progress-bar progress-bar-striped progress-bar-animated"
									role="progressbar"
									:style="{ width: uploadProgress + '%' }"
								></div>
							</div>
							<p>{{ updateStatus }}{{ uploadMessage }}</p>
						</div>
						<div v-else>
							<p>
								<small>Aktuell installierte Version: {{ installed }}</small>
							</p>
							<div v-if="releaseNotes" v-html="releaseNotes"></div>
							<p v-else>
								Keine Releasenotes verfügbar. Mehr Informationen zur neuen Version
								findest du
								<a :href="releaseNotesUrl(available)">hier</a>.
							</p>
						</div>
					</div>
					<div class="modal-footer d-flex justify-content-between">
						<button
							type="button"
							class="btn btn-outline-secondary"
							:disabled="updateStarted"
							data-bs-dismiss="modal"
						>
							Abbrechen
						</button>
						<div>
							<button
								type="button"
								class="btn btn-primary"
								v-if="hasUpdater"
								:disabled="updateStarted"
								@click="update"
							>
								<span v-if="updateStarted">
									<span
										class="spinner-border spinner-border-sm"
										role="status"
										aria-hidden="true"
									>
									</span>
									Akualisieren
								</span>
								<span v-else>Jetzt aktualisieren</span>
							</button>
							<a :href="releaseNotesUrl(available)" class="btn btn-primary" v-else>
								Download
							</a>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import axios from "axios";
import "../icons";

export default {
	name: "Version",
	props: {
		installed: String,
		available: String,
		releaseNotes: String,
		hasUpdater: Boolean,
		uploadMessage: String,
		uploadProgress: Number,
	},
	data: function () {
		return {
			updateStarted: false,
			updateStatus: "",
		};
	},
	methods: {
		update: async function () {
			try {
				await axios.post("update");
				this.updateStatus = "Aktualisierung gestartet: ";
				this.updateStarted = true;
			} catch (e) {
				this.updateStatus = "Aktualisierung nicht möglich: " + e;
			}
		},
		releaseNotesUrl: function (version) {
			return `https://github.com/andig/evcc/releases/tag/${version}`;
		},
	},
	computed: {
		newVersionAvailable: function () {
			return (
				this.available && // available version already computed?
				this.installed != "[[.Version]]" && // go template parsed?
				this.installed != "0.0.1-alpha" && // make used?
				this.available != this.installed
			);
		},
	},
};
</script>

<style scoped>
.icon {
	color: #0fdd42;
}
</style>
