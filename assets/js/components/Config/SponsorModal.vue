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
		:no-buttons="!showForm && false"
		:disable-remove="!showForm"
		disable-cancel
		@changed="$emit('changed')"
		@open="showForm = false"
	>
		<template #default="{ values }">
			<div class="mt-4 mb-3">
				<Sponsor v-bind="sponsor" />
			</div>
			<hr class="my-4" />
			<div v-if="showForm || !token">
				<div class="d-flex justify-content-between align-items-center">
					<p class="fw-bold my-2">Enter your token</p>
					<button
						type="button"
						class="btn btn-link btn-sm text-nowrap text-muted"
						@click="showForm = false"
					>
						{{ $t("config.general.cancel") }}
					</button>
				</div>
				<textarea
					id="sponsorToken"
					v-model="values.token"
					required
					rows="5"
					spellcheck="false"
					class="form-control"
					@paste="(event) => handlePaste(event, values)"
				/>
			</div>
			<div v-if="token">
				<p class="fw-bold my-2">Your token</p>
				<div class="d-flex align-items-start gap-2 text-muted">
					<input
						:value="token"
						disabled
						rows="1"
						class="form-control"
						:class="{ 'is-invalid': error }"
					/>
					<button
						v-if="!fromYaml"
						type="button"
						class="btn btn-link text-nowrap"
						:class="error ? 'text-danger' : 'text-muted'"
						@click="showForm = true"
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
import FormRow from "./FormRow.vue";
import Sponsor, { VICTRON_DEVICE } from "../Savings/Sponsor.vue";
import SponsorTokenExpires from "../Savings/SponsorTokenExpires.vue";
import store from "@/store";
import { docsPrefix } from "@/i18n";
import { cleanYaml } from "@/utils/cleanYaml";
export default {
	name: "SponsorModal",
	components: { FormRow, JsonModal, Sponsor, SponsorTokenExpires },
	props: {
		error: Boolean,
	},
	emits: ["changed"],
	data: () => ({
		showForm: false,
	}),
	computed: {
		sponsor() {
			return store?.state?.sponsor;
		},
		token() {
			return this.sponsor?.token;
		},
		fromYaml() {
			return this.sponsor?.fromYaml;
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
