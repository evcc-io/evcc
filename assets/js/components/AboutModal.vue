<template>
	<GenericModal id="aboutModal" @opened="acknowledge">
		<template #title>
			<a :href="websiteUrl" target="_blank" rel="noopener noreferrer">
				<template v-if="customLogo">
					<img
						:src="'./custom-logo-light'"
						class="about-logo custom-logo custom-logo--light"
						:alt="customBrand || 'evcc'"
					/>
					<img
						:src="'./custom-logo-dark'"
						class="about-logo custom-logo custom-logo--dark"
						:alt="customBrand || 'evcc'"
					/>
				</template>
				<Logo v-else class="about-logo" />
			</a>
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
								<a
									class="text-nowrap"
									:href="nightly ? githubCommitUrl : releaseNotesUrl(installed)"
									target="_blank"
									rel="noopener noreferrer"
								>
									v{{ installed }}
								</a>
								<Badge v-if="newVersionAvailable">
									{{ $t("footer.version.availableLong") }}
								</Badge>
								<Badge v-else-if="!nightly" variant="muted">
									{{ $t("footer.version.latestVersion") }}
								</Badge>
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
					<tr v-if="customPhone">
						<th>{{ $t("footer.version.labelPhone") }}</th>
						<td>
							<a :href="phoneUrl">{{ customPhone }}</a>
						</td>
					</tr>
					<tr v-if="customEmail">
						<th>{{ $t("footer.version.labelEmail") }}</th>
						<td>
							<a :href="`mailto:${customEmail}`">{{ customEmail }}</a>
						</td>
					</tr>
				</tbody>
			</table>

			<!-- changelog -->
			<template v-if="newVersionAvailable">
				<hr />
				<h6>{{ $t("footer.version.modalNextRelease") }}</h6>
				<!-- oxlint-disable vue/no-v-html -->
				<div v-if="releaseNotes" class="release-notes" v-html="cleanedReleaseNotes"></div>
				<!-- oxlint-enable vue/no-v-html -->
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
			<p
				class="mb-0 d-flex justify-content-between align-items-center flex-wrap column-gap-2 row-gap-1"
			>
				<span v-if="customLogo" class="d-flex align-items-center gap-2">
					<span>{{ $t("footer.version.poweredBy") }}</span>
					<a
						:href="evccWebsiteUrl"
						target="_blank"
						rel="noopener noreferrer"
						class="d-flex"
						aria-label="evcc"
					>
						<Logo class="footer-logo" />
					</a>
				</span>
				<i18n-t v-else keypath="footer.version.madeByCommunity" tag="span">
					<a
						:href="githubRepoUrl"
						target="_blank"
						rel="noopener noreferrer"
						class="text-nowrap"
						>{{ $t("footer.version.community") }}</a
					>
				</i18n-t>
				<a
					class="evcc-gray"
					:href="githubDependenciesUrl"
					target="_blank"
					rel="noopener noreferrer"
					>{{ $t("footer.version.openSource") }}</a
				>
			</p>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import GenericModal from "./Helper/GenericModal.vue";
import "@h2d2/shopicons/es/regular/gift";
import "@h2d2/shopicons/es/filled/heart";
import Logo from "./Footer/Logo.vue";
import Badge from "./Helper/Badge.vue";
import api from "@/api";
import settings from "@/settings";
import { extractDomain } from "@/utils/extractDomain";
import { isDevelopment, isNightly, getReleaseName, isNewVersionAvailable } from "@/utils/version";
import { defineComponent } from "vue";

const GITHUB_REPO = "https://github.com/evcc-io/evcc";
const EVCC_WEBSITE = "https://evcc.io/";

export default defineComponent({
	name: "AboutModal",
	components: { GenericModal, Logo, Badge },
	props: {
		installed: { type: String, default: "" },
		commit: { type: String, default: "" },
		availableVersion: String,
		releaseNotes: String,
		hasUpdater: Boolean,
		uploadMessage: String,
		uploadProgress: Number,
		customLogo: Boolean,
		customBrand: String,
		customWebsite: String,
		customEmail: String,
		customPhone: String,
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
			return isNightly(this.installed);
		},
		releaseName() {
			return getReleaseName(this.installed);
		},
		websiteUrl() {
			return this.customWebsite || EVCC_WEBSITE;
		},
		websiteDomain() {
			return extractDomain(this.websiteUrl);
		},
		phoneUrl() {
			// strip separators for RFC 3966 tel: uri
			return `tel:${(this.customPhone || "").replace(/[^+\d]/g, "")}`;
		},
		githubRepoUrl() {
			return GITHUB_REPO;
		},
		evccWebsiteUrl() {
			return EVCC_WEBSITE;
		},
		githubDependenciesUrl() {
			return `${GITHUB_REPO}/network/dependencies`;
		},
		githubCommitUrl() {
			return `${GITHUB_REPO}/commit/${this.commit}`;
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
.footer-logo {
	height: 1rem;
}
.custom-logo {
	height: 4rem;
	width: auto;
	max-width: 100%;
	object-fit: contain;
	object-position: left;
}
.custom-logo--dark {
	display: none;
}
:root.dark .custom-logo--light {
	display: none;
}
:root.dark .custom-logo--dark {
	display: inline;
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
