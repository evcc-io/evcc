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
		<template v-slot:default="{ values }">
			<SponsorTokenExpires v-bind="sponsor" />
			<div class="mt-4 mb-3">
				<Sponsor v-bind="sponsor" />
			</div>
			<button
				v-if="!showForm"
				type="button"
				class="btn btn-link text-muted btn-form"
				@click="showForm = !showForm"
			>
				{{ sponsorTokenLabel }}
			</button>
			<div v-else>
				<hr />
				<FormRow
					id="sponsorToken"
					:label="sponsorTokenLabel"
					:help="
						$t('config.sponsor.descriptionToken', { url: 'https://sponsor.evcc.io/' })
					"
					docs-link="/docs/sponsorship"
					class="mt-4"
				>
					<textarea
						id="sponsorToken"
						v-model="values.token"
						required
						rows="5"
						spellcheck="false"
						class="form-control"
					/>
				</FormRow>
			</div>
		</template>
	</JsonModal>
</template>

<script>
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";
import Sponsor, { VICTRON_DEVICE } from "../Sponsor.vue";
import SponsorTokenExpires from "../SponsorTokenExpires.vue";
import store from "../../store";

export default {
	name: "SponsorModal",
	components: { FormRow, JsonModal, Sponsor, SponsorTokenExpires },
	emits: ["changed"],
	data: () => ({
		showForm: false,
	}),
	methods: {
		transformReadValues() {
			return { token: "" };
		},
	},
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
	},
};
</script>
<style scoped>
textarea {
	font-family: var(--bs-font-monospace);
}
.btn-form {
	margin-left: -0.75rem;
}
</style>
