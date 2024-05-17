<template>
	<GenericModal
		id="sponsorModal"
		ref="modal"
		:title="`${$t('config.sponsor.title')} 💚`"
		data-testid="control-modal"
		@open="open"
	>
		<p>{{ $t("config.sponsor.description") }}</p>
		<p class="text-danger" v-if="error">{{ error }}</p>
		<form ref="form" class="container mx-0 px-0" @submit.prevent="save">
			<FormRow id="sponsorToken" :label="$t('config.sponsor.labelToken')">
				<textarea
					id="sponsorToken"
					v-model="values.interval"
					required
					rows="7"
					spellcheck="false"
					class="form-control"
				/>
			</FormRow>

			<div class="mt-4 d-flex justify-content-end gap-2 flex-column flex-sm-row">
				<button
					type="submit"
					class="btn btn-primary order-1 order-sm-2 flex-grow-1 flex-sm-grow-0 px-4"
					:disabled="saving"
				>
					<span
						v-if="saving"
						class="spinner-border spinner-border-sm"
						role="status"
						aria-hidden="true"
					></span>
					{{ $t("config.general.save") }}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script>
import GenericModal from "./../GenericModal.vue";
import FormRow from "./FormRow.vue";

export default {
	name: "SponsorModal",
	components: { FormRow, GenericModal },
	emits: ["changed"],
	data() {
		return {
			saving: false,
			error: "",
			values: {},
			serverValues: {},
		};
	},
	methods: {
		async open() {
			// open
		},
		async save() {
			alert("save");
		},
	},
};
</script>
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
	padding-right: 0;
}
.btn-cancel {
	margin-left: -0.75rem;
}
textarea {
	font-family: var(--bs-font-monospace);
}
</style>
