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
		:no-buttons="!showForm"
		:disable-remove="!hasToken"
		@changed="$emit('changed')"
		@open="showForm = false"
	>
		<template #default="{ values }">
			<SponsorTokenExpires v-bind="sponsor" />
			<div class="mt-4 mb-3">
				<Sponsor v-bind="sponsor" />
			</div>
			<div v-if="!showForm" class="d-flex gap-1 justify-content-between flex-wrap">
				<button
					type="button"
					class="btn btn-link text-muted text-truncate"
					@click="showForm = !showForm"
				>
					{{ sponsorTokenLabel }}
				</button>
				<a
					v-if="!hasToken"
					class="btn btn-link text-muted text-truncate"
					:href="trialTokenLink"
					target="_blank"
				>
					{{ $t("config.sponsor.trialToken") }}
				</a>
			</div>
			<div v-else>
				<hr />
				<FormRow
					id="sponsorToken"
					:label="sponsorTokenLabel"
					:help="
						$t('config.sponsor.descriptionToken', { url: 'https://sponsor.evcc.io/' })
					"
					docs-link="/docs/sponsorship#trial"
					class="mt-4"
				>
					<textarea
						id="sponsorToken"
						v-model="values.token"
						required
						rows="5"
						spellcheck="false"
						class="form-control"
						@paste="(event) => handlePaste(event, values)"
					/>
				</FormRow>
			</div>
		</template>
	</JsonModal>
</template>

<script>
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";
import Sponsor, { VICTRON_DEVICE } from "../Savings/Sponsor.vue";
import SponsorTokenExpires from "../Savings/SponsorTokenExpires.vue";
import store from "../../store";
import { docsPrefix } from "../../i18n";
import { cleanYaml } from "../../utils/cleanYaml";
export default {
	name: "SponsorModal",
	components: { FormRow, JsonModal, Sponsor, SponsorTokenExpires },
	emits: ["changed"],
	data: () => ({
		showForm: false,
	}),
	computed: {
		sponsor() {
			return store?.state?.sponsor;
		},
		hasToken() {
			const name = this.sponsor?.name || "";
			return name !== "" && name !== VICTRON_DEVICE;
		},
		sponsorTokenLabel() {
			return this.hasToken
				? this.$t("config.sponsor.changeToken")
				: this.$t("config.sponsor.addToken");
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
