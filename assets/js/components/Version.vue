<template>
	<transition name="fade">
		<div v-if="active">
			<div class="row p-3 bg-warning">
				<div class="col-12">
					Neue Version verfügbar! Installiert: {{ installed }}. Verfügbar:
					{{ available }}.
					<b class="px-3" v-if="releaseNotes">
						<a href="#" class="text-body" @click="toggleReleaseNotes">
							Release notes
							<fa-icon
								icon="chevron-down"
								class="expand-icon"
								:class="{ 'expand-icon-rotated': releaseNotesShown }"
							>
							</fa-icon>
						</a>
					</b>
					<b class="px-3">
						<button
							type="button"
							class="btn btn-primary"
							data-toggle="modal"
							data-target="#updateModal"
							v-if="hasUpdater"
							@click="toggleUpdater"
						>
							Aktualisieren
						</button>
						<a
							:href="'https://github.com/andig/evcc/releases/tag/' + available"
							class="text-body"
							v-else
						>
							Download <fa-icon icon="chevron-down"></fa-icon>
						</a>
					</b>
					<button
						type="button"
						class="close float-right"
						style="margin-top: -2px"
						aria-label="Close"
						@click="dismiss"
					>
						<span aria-hidden="true">&times;</span>
					</button>
				</div>
			</div>

			<transition name="fade">
				<div class="row p-3 bg-light" v-if="releaseNotesShown">
					<div class="col-12" v-html="releaseNotes"></div>
				</div>
			</transition>

			<transition name="display">
				<div
					id="updateModal"
					class="dialog"
					tabindex="-1"
					role="dialog"
					v-if="updaterShown"
				>
					<div class="modal-dialog modal-dialog-centered" role="document">
						<div class="modal-content">
							<div class="modal-header">
								<h4 class="modal-title font-weight-bold">
									Aktualisierung durchführen
								</h4>
								<button type="button" class="close" @click="toggleUpdater">
									<span aria-hidden="true">&times;</span>
								</button>
							</div>
							<div class="modal-body">
								<p class="font-weight-bold">
									Nach Aktualisierung wird evcc neu gestartet.
								</p>
								<div
									class="progress"
									style="margin-top: 16px; margin-bottom: 16px"
									v-if="updateStarted"
								>
									<div
										class="progress-bar"
										role="progressbar"
										:style="{ width: uploadProgress + '%' }"
									></div>
								</div>
								<p>{{ updateStatus }}{{ uploadMessage }}</p>
							</div>
							<div class="modal-footer">
								<button
									type="button"
									class="btn btn-secondary"
									:class="{ disabled: updateStarted }"
									@click="toggleUpdater"
								>
									Abbrechen
								</button>
								<button type="button" class="btn btn-danger" @click="update">
									Installieren
								</button>
							</div>
						</div>
					</div>
				</div>
			</transition>
		</div>
	</transition>
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
			dismissed: false,
			releaseNotesShown: false,
			updaterShown: false,
			updateStarted: false,
			updateStatus: "",
		};
	},
	methods: {
		dismiss: function () {
			this.dismissed = true;
		},
		toggleReleaseNotes: function (e) {
			e.preventDefault();
			this.releaseNotesShown = !this.releaseNotesShown;
		},
		toggleUpdater: function (e) {
			e.preventDefault();
			if (!this.updateStarted) {
				this.updaterShown = !this.updaterShown;
			}
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
	},
	computed: {
		active: function () {
			return (
				this.available && // available version already computed?
				this.installed != "[[.Version]]" && // go template parsed?
				this.installed != "0.0.1-alpha" && // make used?
				this.available != this.installed &&
				this.dismissed === false
			);
		},
	},
	watch: {
		available: function () {
			this.dismissed = false;
			this.releaseNotesShown = false;
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
.display-enter-active {
	display: block !important;
}

.expand-icon {
	transition: transform 0.25s ease-in;
	transform: rotate(0);
}
.expand-icon-rotated {
	transform: rotate(-180deg);
}
</style>
