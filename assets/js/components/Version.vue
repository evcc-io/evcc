<template>
	<div>
		<small class="text-black">
			<a href="#" @click.prevent="showModal" v-if="newVersionAvailable">
				<fa-icon icon="gift" class="icon mr-1"></fa-icon>Update<span
					class="d-none d-sm-inline"
				>
					verfügbar</span
				>:
				{{ available }}
			</a>
			<a :href="releaseNotesUrl(installed)" target="_blank" v-else>
				Version {{ installed }}
			</a>
		</small>

		<transition name="fade">
			<div id="updateModal" class="dialog" tabindex="-1" role="dialog" v-if="modalActive">
				<div
					class="modal-dialog modal-dialog-centered modal-dialog-scrollable"
					role="document"
				>
					<div class="modal-content">
						<div class="modal-header">
							<h4 class="modal-title font-weight-bold">Update verfügbar</h4>
							<button
								type="button"
								class="close"
								:disabled="updateStarted"
								@click="closeModal"
							>
								<span aria-hidden="true">&times;</span>
							</button>
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
									Keine Releasenotes verfügbar. Mehr Informationen zur neuen
									Version findest du
									<a :href="releaseNotesUrl(available)">hier</a>.
								</p>
							</div>
						</div>
						<div class="modal-footer d-flex justify-content-between">
							<button
								type="button"
								class="btn btn-outline-secondary"
								:disabled="updateStarted"
								@click="closeModal"
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
								<a
									:href="releaseNotesUrl(available)"
									class="btn btn-primary"
									v-else
								>
									Download
								</a>
							</div>
						</div>
					</div>
				</div>
			</div>
		</transition>
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
			modalActive: false,
			updateStarted: false,
			updateStatus: "",
		};
	},
	methods: {
		showModal: function () {
			this.modalActive = true;
		},
		closeModal: function () {
			this.modalActive = false;
		},
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
.fade-enter-active,
.fade-leave-active {
	transition: opacity 0.25s ease-in;
}
.fade-enter,
.fade-leave-to {
	opacity: 0;
}
.dialog {
	position: fixed;
	top: 0;
	left: 0;
	z-index: 1050;
	width: 100%;
	height: 100%;
	overflow: hidden;
	outline: 0;
}
.icon {
	color: #0fdd42;
}
.text-black a {
	color: #18191a;
}
</style>
