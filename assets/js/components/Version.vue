<template>
	<div>
		<button
			v-if="newVersionAvailable"
			href="#"
			data-bs-toggle="modal"
			data-bs-target="#updateModal"
			class="btn btn-link ps-0 text-decoration-none text-white text-nowrap"
		>
			<fa-icon icon="gift" class="icon me-2"></fa-icon>v{{ installed }}
			<span class="d-none d-xs-inline d-sm-none text-white text-decoration-underline">
				{{ $t("footer.version.availableShort") }}
			</span>
			<span class="d-none d-sm-inline text-muted text-white text-decoration-underline">
				{{ $t("footer.version.availableLong") }}
			</span>
		</button>
		<a
			v-else
			:href="releaseNotesUrl(installed)"
			target="_blank"
			class="btn btn-link text-white ps-0 text-decoration-none text-nowrap"
		>
			<span class="d-inline d-xs-none d-sm-none">{{
				$t("footer.version.versionShort", { installed })
			}}</span>
			<span class="d-none d-xs-inline d-sm-inline">{{
				$t("footer.version.versionLong", { installed })
			}}</span>
		</a>

		<div
			id="updateModal"
			class="modal fade text-dark"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">{{ $t("footer.version.modalTitle") }}</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<div v-if="updateStarted">
							<p>{{ $t("footer.version.modalUpdateStarted") }}</p>
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
								<small>
									{{ $t("footer.version.modalInstalledVersion") }}:
									{{ installed }}
								</small>
							</p>
							<!-- eslint-disable vue/no-v-html -->
							<div v-if="releaseNotes" v-html="releaseNotes"></div>
							<!-- eslint-enable vue/no-v-html -->
							<p v-else>
								{{ $t("footer.version.modalNoReleaseNotes") }}
								<a :href="releaseNotesUrl(available)">GitHub</a>.
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
							{{ $t("footer.version.modalCancel") }}
						</button>
						<div>
							<button
								v-if="hasUpdater"
								type="button"
								class="btn btn-primary"
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
									{{ $t("footer.version.modalUpdate") }}
								</span>
								<span v-else>{{ $t("footer.version.modalUpdateNow") }}</span>
							</button>
							<a v-else :href="releaseNotesUrl(available)" class="btn btn-primary">
								{{ $t("footer.version.modalDownload") }}
							</a>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import api from "../api";
import "../icons";

export default {
	name: "Version",
	props: {
		installed: String,
		available: String,
		releaseNotes: String,
		commit: String,
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
	computed: {
		githubHashUrl: function () {
			return `https://github.com/evcc-io/evcc/commit/${this.commit}`;
		},
		newVersionAvailable: function () {
			return (
				this.available && // available version already computed?
				this.installed != "[[.Version]]" && // go template parsed?
				this.installed != "0.0.1-alpha" && // make used?
				this.available != this.installed
			);
		},
	},
	methods: {
		update: async function () {
			try {
				await api.post("update");
				this.updateStatus = this.$t("footer.version.modalUpdateStatusStart");
				this.updateStarted = true;
			} catch (e) {
				this.updateStatus = this.$t("footer.version.modalUpdateStatusStart") + e;
			}
		},
		releaseNotesUrl: function (version) {
			return `https://github.com/evcc-io/evcc/releases/tag/${version}`;
		},
	},
};
</script>

<style scoped>
.icon {
	color: var(--evcc-dark-green);
}
</style>
