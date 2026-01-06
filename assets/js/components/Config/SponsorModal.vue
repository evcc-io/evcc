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
		@open="showForm = false"
	>
		<template #default="{ values }">
			<div class="mt-4 mb-3">
				<Sponsor v-bind="sponsor" />
			</div>
			<hr class="my-4" />
			<div v-if="showTokenForm">
				<div class="d-flex justify-content-between align-items-center">
					<label for="sponsorToken" class="fw-bold my-2">{{
						$t("config.sponsor.enterYourToken")
					}}</label>
					<button
						v-if="hasUiToken"
						type="button"
						class="btn btn-link btn-sm text-nowrap text-muted"
						@click="
							editMode = false;
							values.token = '';
						"
					>
						{{ $t("config.general.cancel") }}
					</button>
				</div>
				<textarea
					id="sponsorToken"
					v-model="values.token"
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
			<div v-else-if="token">
				<label for="existingToken" class="fw-bold my-2">{{
					$t("config.sponsor.yourToken")
				}}</label>
				<div class="d-flex align-items-start gap-2 text-muted">
					<input
						id="existingToken"
						:value="token"
						disabled
						rows="1"
						class="form-control"
						:class="{ 'is-invalid': error }"
					/>
					<span v-if="fromYaml" class="text-muted text-nowrap align-self-center ms-2">
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
	}),
	computed: {
		sponsor() {
			return store?.state?.sponsor;
		},
		token() {
			return this.sponsor?.status.token;
		},
		fromYaml() {
			return this.sponsor?.fromYaml;
		},
		name() {
			return this.sponsor?.status.name || "";
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
			return { token: "" };
		},
		handlePaste(event, values) {
			event.preventDefault();
			const text = event.clipboardData.getData("text");
			const cleaned = cleanYaml(text, "sponsortoken");
			values.token = cleaned;
		},
	},
};
</script>
<style scoped>
textarea {
	font-family: var(--bs-font-monospace);
}
</style>
