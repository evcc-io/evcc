<template>
	<GenericModal id="aboutModal" :size="modalSize" @opened="acknowledge">
		<template #title>
			<a :href="websiteUrl" target="_blank" rel="noopener noreferrer"
				><Logo class="about-logo"
			/></a>
		</template>
		<div v-if="updateStarted">
			<p>{{ $t("footer.version.modalUpdateStarted") }}</p>
			<div class="progress my-3">
				<div
					class="progress-bar progress-bar-striped progress-bar-animated"
					role="progressbar"
					:style="{ width: uploadProgress + '%' }"
				></div>
			</div>
			<p>{{ updateStatus }} {{ uploadMessage }}</p>
		</div>
		<div v-else>
			<table class="about-table">
				<tbody>
					<tr>
						<th>{{ $t("footer.version.labelVersion") }}</th>
						<td v-if="development">---</td>
						<td v-else>
							<div class="d-flex flex-wrap column-gap-2 align-items-baseline">
								<span class="text-nowrap">
									<a
										:href="releaseNotesUrl(installed)"
										target="_blank"
										rel="noopener noreferrer"
									>
										v{{ installed }}
									</a>
									<template v-if="nightly">
										(<a
											:href="githubCommitUrl"
											target="_blank"
											rel="noopener noreferrer"
											><span class="font-monospace">{{
												shortCommit
											}}</span></a
										>)
									</template>
								</span>
								<span
									v-if="!nightly && !newVersionAvailable"
									class="text-muted text-nowrap"
									>{{ $t("footer.version.latestVersion") }}</span
								>
								<span v-if="newVersionAvailable" class="text-nowrap">{{
									$t("footer.version.availableLong")
								}}</span>
							</div>
						</td>
					</tr>
					<tr>
						<th>{{ $t("footer.version.labelRelease") }}</th>
						<td>{{ releaseName }}</td>
					</tr>
					<tr>
						<th>{{ $t("footer.version.labelWebsite") }}</th>
						<td>
							<a :href="websiteUrl" target="_blank" rel="noopener noreferrer">
								{{ websiteDomain }}
							</a>
						</td>
					</tr>
				</tbody>
			</table>

			<!-- changelog -->
			<template v-if="newVersionAvailable">
				<hr />
				<h6>{{ $t("footer.version.modalNextRelease") }}</h6>
				<!-- eslint-disable vue/no-v-html -->
				<div v-if="releaseNotes" class="release-notes" v-html="cleanedReleaseNotes"></div>
				<!-- eslint-enable vue/no-v-html -->
				<p v-else>
					{{ $t("footer.version.modalNoReleaseNotes") }}
					<a :href="releaseNotesUrl(availableVersion)">GitHub</a>.
				</p>
			</template>

			<!-- update actions -->
			<template v-if="newVersionAvailable">
				<div class="d-flex justify-content-end mt-3">
					<button v-if="hasUpdater" type="button" class="btn btn-primary" @click="update">
						{{ $t("footer.version.modalUpdateNow") }}
					</button>
					<a
						v-else
						:href="releaseNotesUrl(availableVersion)"
						target="_blank"
						rel="noopener noreferrer"
						class="btn btn-outline-primary"
					>
						{{ $t("footer.version.modalViewOnGitHub") }}
					</a>
				</div>
			</template>

			<!-- open source -->
			<hr />
			<p class="mb-0 small d-flex flex-wrap column-gap-1">
				<i18n-t keypath="footer.version.madeByCommunity" tag="span">
					<a
						:href="githubRepoUrl"
						target="_blank"
						rel="noopener noreferrer"
						class="text-nowrap"
						>{{ $t("footer.version.community") }}</a
					>
				</i18n-t>
				<i18n-t keypath="footer.version.poweredByOpenSource" tag="span" class="d-inline">
					<a
						class="text-muted"
						:href="githubDependenciesUrl"
						target="_blank"
						rel="noopener noreferrer"
						>{{ $t("footer.version.openSource") }}</a
					>
				</i18n-t>
			</p>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import GenericModal from "./Helper/GenericModal.vue";
import "@h2d2/shopicons/es/regular/gift";
import "@h2d2/shopicons/es/filled/heart";
import Logo from "./Footer/Logo.vue";
import api from "@/api";
import settings from "@/settings";
import { extractDomain } from "@/utils/extractDomain";
import {
	isDevelopment,
	isNightly,
	getReleaseName,
	shortCommit,
	isNewVersionAvailable,
} from "@/utils/version";
import { defineComponent } from "vue";

const GITHUB_REPO = "https://github.com/evcc-io/evcc";
const EVCC_WEBSITE = "https://evcc.io/";

export default defineComponent({
	name: "AboutModal",
	components: { GenericModal, Logo },
	props: {
		installed: { type: String, default: "" },
		commit: String,
		availableVersion: String,
		releaseNotes: String,
		hasUpdater: Boolean,
		uploadMessage: String,
		uploadProgress: Number,
	},
	data() {
		return {
			updateStarted: false,
			updateStatus: "",
		};
	},
	computed: {
		development() {
			return isDevelopment(this.installed);
		},
		nightly() {
			return isNightly(this.installed, this.commit);
		},
		releaseName() {
			return getReleaseName(this.installed, this.commit);
		},
		websiteUrl() {
			return EVCC_WEBSITE;
		},
		websiteDomain() {
			return extractDomain(EVCC_WEBSITE);
		},
		githubRepoUrl() {
			return GITHUB_REPO;
		},
		githubDependenciesUrl() {
			return `${GITHUB_REPO}/network/dependencies`;
		},
		shortCommit() {
			return shortCommit(this.commit);
		},
		githubCommitUrl() {
			return `${GITHUB_REPO}/commit/${this.commit}`;
		},
		modalSize() {
			return this.newVersionAvailable ? undefined : "sm";
		},
		cleanedReleaseNotes() {
			if (!this.releaseNotes) return "";
			return this.releaseNotes.replaceAll("<h2>Changelog</h2>", "");
		},
		newVersionAvailable() {
			return isNewVersionAvailable(this.installed, this.availableVersion);
		},
	},
	methods: {
		async update() {
			try {
				await api.post("update");
				this.updateStatus = this.$t("footer.version.modalUpdateStatusStart");
				this.updateStarted = true;
			} catch (e) {
				this.updateStatus = `${this.$t("footer.version.modalUpdateStatusStart")} ${e}`;
			}
		},
		releaseNotesUrl(version?: string) {
			return `${GITHUB_REPO}/releases/tag/${version}`;
		},
		acknowledge() {
			if (!this.newVersionAvailable) return;
			settings.lastAcknowledgedVersion = this.availableVersion!;
		},
	},
});
</script>

<style scoped>
.about-logo {
	height: 2.5rem;
}
.about-table th {
	padding-right: 1rem;
	font-weight: normal;
	vertical-align: top;
}
.about-table td {
	vertical-align: top;
}
.release-notes :deep(h1) {
	font-size: 1.5rem;
	font-weight: bold;
	margin: 2rem 0 1rem;
	text-transform: none;
}
.release-notes :deep(h2) {
	font-size: 1.25rem;
	font-weight: bold;
}
.release-notes :deep(h3) {
	font-size: 1rem;
	font-weight: bold;
}
.release-notes :deep(h1:first-child) {
	margin-top: 0;
}
</style>
