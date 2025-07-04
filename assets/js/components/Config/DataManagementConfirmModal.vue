<template>
	<GenericModal
		id="dataManagementConfirmModal"
		:title="$t('config.system.dataManagement.confirmWithPassword')"
		size="md"
		data-testid="data-management-confirm-modal"
		@close="$emit('close')"
		@closed="closed"
	>
		<form @submit.prevent="submit">
			<p>
				<span>{{ $t(`config.system.dataManagement.${type}.confirmationText`) }}</span>
			</p>

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
import { defineComponent, type PropType } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import PasswordInput from "../Auth/PasswordInput.vue";
import { downloadFile } from "@/api";
import { isLoggedIn } from "../Auth/auth";
import Modal from "bootstrap/js/dist/modal";
import type { ConfirmAction } from "@/types/evcc";

export default defineComponent({
	name: "DataManagementConfirmModal",
	components: { GenericModal, PasswordInput },
	props: {
		type: { type: String as PropType<"backup" | "restore"> },
		action: { type: Function as PropType<ConfirmAction> },
	},
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
				if (this.action) {
					const res = await this.action(this.password);

					if (res.status === 200) {
						if (isLoggedIn()) {
							this.closeModal();

							if (this.type === "backup") {
								downloadFile(res);
							}
						} else {
							this.iframeHint = true;
						}
					} else if (res.status === 401) {
						this.error = this.$t("loginModal.invalid");
					}
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
