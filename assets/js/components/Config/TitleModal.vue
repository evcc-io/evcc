<template>
	<GenericModal
		id="titleModal"
		ref="modal"
		:title="$t('config.title.title')"
		data-testid="title-modal"
	>
		<p class="text-danger" v-if="error">{{ error }}</p>
		<form ref="form" class="container mx-0 px-0">
			<FormRow
				id="siteTitle"
				:label="$t('config.title.label')"
				:help="$t('config.title.description')"
			>
				<input id="siteTitle" v-model="title" class="form-control" />
			</FormRow>
			<div class="my-4 d-flex justify-content-between">
				<button type="button" class="btn btn-link text-muted" data-bs-dismiss="modal">
					{{ $t("config.general.cancel") }}
				</button>
				<button
					type="submit"
					class="btn btn-primary"
					:disabled="saving"
					@click.prevent="save"
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
import GenericModal from "../GenericModal.vue";
import FormRow from "./FormRow.vue";
import api from "../../api";

export default {
	name: "TitleModal",
	components: { FormRow, GenericModal },
	emits: ["changed"],
	data() {
		return {
			saving: false,
			error: "",
			title: "",
		};
	},
	async mounted() {
		await this.load();
	},
	methods: {
		async load() {
			try {
				const { data } = await api.get("/config/site");
				this.title = data.result.title;
			} catch (e) {
				console.error(e);
			}
		},
		async save() {
			this.saving = true;
			this.error = "";
			try {
				await api.put("/config/site", { title: this.title });
				this.$emit("changed");
				this.$refs.modal.close();
			} catch (e) {
				console.error(e);
				this.error = e.response.data.error;
			}
			this.saving = false;
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
</style>
