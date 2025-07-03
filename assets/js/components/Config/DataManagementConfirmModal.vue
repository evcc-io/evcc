<template>
	<GenericModal
		id="dataManagementConfirmModal"
		:title="$t('config.system.dataManagement.confirmWithPassword')"
		size="sm"
		data-testid="data-management-confirm-modal"
		@close="$emit('close')"
		@closed="closed"
	>
		<form @submit.prevent="submit">
			<PasswordInput v-model:password="password" :error="error" :iframe-hint="iframeHint" />

			<button type="submit" class="btn btn-primary w-100 mb-3" :disabled="loading">
				<span
					v-if="loading"
					class="spinner-border spinner-border-sm"
					role="status"
					aria-hidden="true"
				></span>
				{{ $t("loginModal.login") }}
			</button>
		</form>
	</GenericModal>
</template>
<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import PasswordInput from "../Auth/PasswordInput.vue";
import api, { downloadFile } from "@/api";
import { isLoggedIn } from "../Auth/auth";
import Modal from "bootstrap/js/dist/modal";

export default defineComponent({
	name: "DataManagementConfirmModal",
	components: { GenericModal, PasswordInput },
	emits: ["close"],
	data() {
		return {
			password: "",
			loading: false,
			iframeHint: false,
			error: "",
		};
	},
	methods: {
		closed() {
			this.password = "";
			this.loading = false;
			this.iframeHint = false;
			this.error = "";
		},
		closeModal() {
			Modal.getOrCreateInstance(
				document.getElementById("dataManagementConfirmModal") as HTMLElement
			).hide();
		},
		async submit() {
			this.loading = true;

			try {
				const res = await api.post(
					"/config/backup",
					{ password: this.password },
					{
						responseType: "blob",
						validateStatus: (code: number) => [200, 401, 403].includes(code),
					}
				);

				if (res.status === 200) {
					if (isLoggedIn()) {
						this.closeModal();

						downloadFile(res);
					} else {
						this.iframeHint = true;
					}
				} else if (res.status === 401) {
					this.error = this.$t("loginModal.invalid");
				}
			} catch (err) {
				console.error(err);
				if (err instanceof Error) {
					this.error = err.message;
				}
			} finally {
				this.loading = false;
			}
		},
	},
});
</script>
