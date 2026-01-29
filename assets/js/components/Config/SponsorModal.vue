<template>
	<JsonModal
		id="sponsorModal"
		:title="`${$t('config.sponsor.title')} 💚`"
		:description="$t('config.sponsor.description')"
		:error-message="$t('config.sponsor.error')"
		endpoint="/config/sponsortoken"
		:transform-read-values="transformReadValues"
		data-testid="sponsor-modal"
		size="lg"
		:no-buttons="notUiEditable"
		:disable-remove="!hasUiToken"
		disable-cancel
		@changed="$emit('changed')"
		@open="
			showForm = false;
			activeTab = 'github';
		"
	>
		<template #default="{ values }">
			<div class="mt-4 mb-3">
				<Sponsor v-bind="sponsor" />
			</div>
			<hr class="my-4" />
			<div v-if="showTokenForm">
				<ul v-if="$hiddenFeatures()" class="nav nav-tabs mb-3" role="tablist">
					<li class="nav-item" role="presentation">
						<button
							class="nav-link"
							:class="{ active: activeTab === 'github' }"
							type="button"
							@click="
								if (activeTab !== 'github') {
									values.token = '';
									values.email = '';
								}
								activeTab = 'github';
							"
						>
							{{ $t("config.sponsor.tabGitHub") }}
						</button>
					</li>
					<li class="nav-item" role="presentation">
						<button
							class="nav-link"
							:class="{ active: activeTab === 'creem' }"
							type="button"
							@click="
								if (activeTab !== 'creem') {
									values.token = '';
									values.email = '';
								}
								activeTab = 'creem';
							"
						>
							{{ $t("config.sponsor.tabCreem") }} 🧪
						</button>
					</li>
				</ul>
				<div v-if="activeTab === 'github'">
					<label for="sponsorToken" class="my-2">
						{{ $t("config.sponsor.enterYourToken") }}
					</label>
					<textarea
						id="sponsorToken"
						v-model.trim="values.token"
						class="form-control mb-1"
						required
						rows="5"
						spellcheck="false"
						@paste="(event) => handlePaste(event, values)"
					/>
					<i18n-t tag="small" keypath="config.sponsor.descriptionToken" scope="global">
						<template #url>
							<a href="https://sponsor.evcc.io" target="_blank">sponsor.evcc.io</a>
						</template>
						<template #trialToken>
							<a :href="trialTokenLink" target="_blank">{{
								$t("config.sponsor.trialToken")
							}}</a>
						</template>
					</i18n-t>
				</div>
				<div v-else-if="activeTab === 'creem'">
					<label for="sponsorEmail" class="my-2">{{ $t("config.sponsor.email") }}</label>
					<input
						id="sponsorEmail"
						v-model.trim="values.email"
						type="email"
						class="form-control mb-1"
						required
					/>
					<i18n-t
						tag="small"
						keypath="config.sponsor.emailHint"
						scope="global"
						class="mb-3 d-block text-muted"
					>
						<template #url>
							<a href="https://sponsor.evcc.io/#creem" target="_blank"
								>direct sponsoring</a
							>
						</template>
					</i18n-t>
					<label for="licenseKey" class="my-2">{{
						$t("config.sponsor.activationKey")
					}}</label>
					<input
						id="licenseKey"
						v-model.trim="values.token"
						class="form-control mb-1 font-monospace"
						required
						spellcheck="false"
						pattern="[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}"
						title="XXXXX-XXXXX-XXXXX-XXXXX-XXXXX"
					/>
					<i18n-t tag="small" keypath="config.sponsor.activationKeyHint" scope="global">
						<template #url>
							<a href="https://www.creem.io/portal/" target="_blank">Creem Portal</a>
						</template>
					</i18n-t>
				</div>
				<div v-if="hasUiToken" class="d-flex justify-content-end mt-3">
					<button
						type="button"
						class="btn btn-link text-nowrap text-muted"
						@click="
							editMode = false;
							values.token = '';
							values.email = '';
						"
					>
						{{ $t("config.general.cancel") }}
					</button>
				</div>
			</div>
			<div v-else-if="token">
				<div v-if="activationKey">
					<label for="existingEmail" class="fw-bold my-2">{{
						$t("config.sponsor.email")
					}}</label>
					<div class="text-muted mb-3">
						<input id="existingEmail" :value="name" disabled class="form-control" />
					</div>
					<label for="existingActivationKey" class="fw-bold my-2">{{
						$t("config.sponsor.activationKey")
					}}</label>
					<div class="text-muted">
						<input
							id="existingActivationKey"
							:value="activationKey"
							disabled
							class="form-control font-monospace"
							:class="{ 'is-invalid': error }"
						/>
					</div>
					<div class="d-flex justify-content-end mt-2">
						<button
							type="button"
							class="btn btn-link text-nowrap"
							:class="error ? 'text-danger' : 'text-muted'"
							@click="editMode = true"
						>
							{{ $t("config.general.change") }}
						</button>
					</div>
				</div>
				<div v-else>
					<label for="existingToken" class="fw-bold my-2">{{
						$t("config.sponsor.yourToken")
					}}</label>
					<div class="text-muted">
						<input
							id="existingToken"
							:value="token"
							disabled
							rows="1"
							class="form-control"
							:class="{ 'is-invalid': error }"
						/>
					</div>
					<div class="d-flex justify-content-end mt-2">
						<span v-if="fromYaml" class="text-nowrap small text-muted">
							{{ $t("config.sponsor.viaYaml") }}
						</span>
						<button
							v-else
							type="button"
							class="btn btn-link text-nowrap"
							:class="error ? 'text-danger' : 'text-muted'"
							@click="editMode = true"
						>
							{{ $t("config.sponsor.changeToken") }}
						</button>
					</div>
				</div>
				<SponsorTokenExpires v-bind="sponsor" />
			</div>
		</template>
	</JsonModal>
</template>

<script>
import JsonModal from "./JsonModal.vue";
import Sponsor from "../Savings/Sponsor.vue";
import SponsorTokenExpires from "../Savings/SponsorTokenExpires.vue";
import store from "@/store";
import { docsPrefix } from "@/i18n";
import { cleanYaml } from "@/utils/cleanYaml";
export default {
	name: "SponsorModal",
	components: { JsonModal, Sponsor, SponsorTokenExpires },
	props: {
		error: Boolean,
	},
	emits: ["changed"],
	data: () => ({
		editMode: false,
		activeTab: "github",
	}),
	computed: {
		sponsor() {
			return store?.state?.sponsor;
		},
		token() {
			return this.sponsor?.status?.token;
		},
		activationKey() {
			return this.sponsor?.status?.activationKey;
		},
		fromYaml() {
			return this.sponsor?.fromYaml;
		},
		name() {
			return this.sponsor?.status?.name || "";
		},
		showTokenForm() {
			return this.editMode || !this.token;
		},
		notUiEditable() {
			return !!this.name && this.fromYaml;
		},
		hasUiToken() {
			return this.token && !this.fromYaml;
		},
		trialTokenLink() {
			return `${docsPrefix()}/docs/sponsorship#trial`;
		},
	},
	methods: {
		transformReadValues() {
			return { token: "", email: "" };
		},
		handlePaste(event, values) {
			event.preventDefault();
			const text = event.clipboardData.getData("text");
			const cleaned = cleanYaml(text, "sponsortoken");
			values.token = cleaned;
			if (this.activeTab === "github" && this.isLicenseKey(cleaned)) {
				this.activeTab = "creem";
			}
		},
		isLicenseKey(token) {
			// Match pattern XXXXX-XXXXX-XXXXX-XXXXX-XXXXX (case-insensitive alphanumeric)
			const pattern = /^[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}$/i;
			return pattern.test(token || "");
		},
	},
};
</script>
<style scoped>
textarea {
	font-family: var(--bs-font-monospace);
}
</style>
