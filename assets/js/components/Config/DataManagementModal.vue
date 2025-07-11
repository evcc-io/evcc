<template>
	<div>
		<GenericModal
			id="dataManagementModal"
			:title="$t('config.system.dataManagement.title')"
			data-testid="data-management-modal"
			@closed="dataManagementModalClosed"
		>
			<p>
				<span>{{ $t("config.system.dataManagement.description") }}</span>
			</p>

			<div class="mb-3">
				<h6>
					{{ $t("config.system.dataManagement.backup.title") }}
				</h6>

				<p>
					{{ $t("config.system.dataManagement.backup.description") }}
				</p>
				<button
					class="btn btn-outline-secondary"
					@click="openDataManagementConfirmModal('backup')"
				>
					{{ $t("config.system.dataManagement.backup.action") }}
				</button>
			</div>
			<div class="mb-3">
				<h6>
					{{ $t("config.system.dataManagement.restore.title") }}
				</h6>
				<p>
					{{ $t("config.system.dataManagement.restore.description") }}
				</p>

				<form @submit.prevent="openDataManagementConfirmModal('restore')">
					<FormRow
						id="restoreFile"
						:label="$t('config.system.dataManagement.restore.labelFile')"
					>
						<PropertyFileField
							id="restoreFile"
							ref="fileInput"
							:accepted="['.db']"
							required
							@file-changed="fileChanged"
						/>
					</FormRow>

					<button class="btn btn-outline-danger mt-2" :disabled="file === null">
						{{ $t("config.system.dataManagement.restore.action") }}
					</button>
				</form>
			</div>
			<div class="mb-3">
				<h6>
					{{ $t("config.system.dataManagement.reset.title") }}
				</h6>
				<p>{{ $t("config.system.dataManagement.reset.description") }}</p>

				<form @submit.prevent="openDataManagementConfirmModal('reset')">
					<div class="d-flex flex-column mb-2">
						<div class="d-flex mb-1">
							<input
								id="resetSessions"
								v-model="selectedReset.sessions"
								class="form-check-input"
								type="checkbox"
							/>
							<label class="form-check-label ms-2" for="resetSessions">
								<span>{{ $t("config.system.dataManagement.reset.sessions") }}</span>
								<br />
								<small>
									{{
										$t("config.system.dataManagement.reset.sessionsDescription")
									}}
								</small>
							</label>
						</div>

						<div class="d-flex mb-1">
							<input
								id="resetSettings"
								v-model="selectedReset.settings"
								class="form-check-input"
								type="checkbox"
							/>
							<label class="form-check-label ms-2" for="resetSettings">
								<span>{{ $t("config.system.dataManagement.reset.settings") }}</span>
								<br />
								<small>
									{{
										$t("config.system.dataManagement.reset.settingsDescription")
									}}
								</small>
							</label>
						</div>
					</div>

					<button
						class="btn btn-outline-danger mt-3"
						:disabled="!selectedReset.sessions && !selectedReset.settings"
					>
						{{ $t("config.system.dataManagement.reset.action") }}
					</button>
				</form>
			</div>
			<p>
				<small>
					{{ $t("config.system.dataManagement.note") }}
				</small>
			</p>
		</GenericModal>
		<GenericModal
			id="dataManagementConfirmModal"
			:title="$t(`config.system.dataManagement.${confirmType}.title`)"
			size="md"
			data-testid="data-management-confirm-modal"
			@close="confirmModalClose"
			@closed="confirmType = ''"
		>
			<form @submit.prevent="submit">
				<p>
					<span>{{
						$t(`config.system.dataManagement.${confirmType}.confirmationText`)
					}}</span>
				</p>

				<PasswordInput
					v-model:password="password"
					:error="error"
					:iframe-hint="iframeHint"
					:required="false"
				/>

				<div class="d-flex justify-content-between">
					<button
						:disabled="loading"
						type="button"
						class="btn btn-outline-secondary"
						@click="closeDataManagementConfirmModal"
					>
						<span>{{ $t(`config.system.dataManagement.cancel`) }}</span>
					</button>

					<button
						type="submit"
						class="btn"
						:class="confirmType === 'backup' ? 'btn-primary' : 'btn-danger'"
						:disabled="loading"
					>
						<span
							v-if="loading"
							class="spinner-border spinner-border-sm"
							role="status"
							aria-hidden="true"
						></span>
						{{ $t(`config.system.dataManagement.${confirmType}.confirmationButton`) }}
					</button>
				</div>
			</form>
		</GenericModal>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import api, { downloadFile } from "@/api";
import PropertyFileField from "./PropertyFileField.vue";
import FormRow from "./FormRow.vue";
import { isLoggedIn } from "../Auth/auth";
import type { AxiosResponse } from "axios";
import Modal from "bootstrap/js/dist/modal";
import PasswordInput from "../Auth/PasswordInput.vue";
import restart, { showRestarting } from "@/restart";

const validateStatus = (code: number) => [200, 204, 401, 403].includes(code);

export default defineComponent({
	name: "DataManagementModal",
	components: { GenericModal, PropertyFileField, FormRow, PasswordInput },
	data() {
		return {
			selectedReset: {
				sessions: false,
				settings: false,
			},
			file: null as File | null,
			confirmType: "" as "backup" | "restore" | "reset" | "",
			password: "",
			loading: false,
			iframeHint: false,
			error: "",
			hideDataManagementModal: false,
			navigateHomeAfterRestart: false,
		};
	},
	computed: {
		restarting() {
			return restart.restarting;
		},
	},
	watch: {
		restarting(newVal) {
			// wait for restarte before navigate (ensure lazy chunks are available)
			if (!newVal && this.navigateHomeAfterRestart) {
				this.$router.push({ path: "/" });
				this.navigateHomeAfterRestart = false;
			}
		},
	},
	methods: {
		resetDataManagementConfirmModal() {
			this.password = "";
			this.loading = false;
			this.iframeHint = false;
			this.error = "";
		},
		resetDataManagementModal() {
			this.selectedReset = {
				sessions: false,
				settings: false,
			};
			this.file = null;
			this.navigateHomeAfterRestart = false;
			(
				this.$refs["fileInput"] as InstanceType<typeof PropertyFileField> | undefined
			)?.reset();
		},
		reset() {
			this.resetDataManagementConfirmModal();
			this.resetDataManagementModal();
			this.dataManagementConfirmModal().hide();
			this.dataManagementModal().hide();
		},
		dataManagementModal() {
			return Modal.getOrCreateInstance(
				document.getElementById("dataManagementModal") as HTMLElement
			);
		},
		dataManagementConfirmModal() {
			return Modal.getOrCreateInstance(
				document.getElementById("dataManagementConfirmModal") as HTMLElement
			);
		},
		openDataManagementConfirmModal(type: typeof this.confirmType) {
			this.resetDataManagementConfirmModal();
			this.dataManagementConfirmModal().show();
			this.dataManagementModal().hide();
			this.confirmType = type;
		},
		closeDataManagementConfirmModal() {
			this.dataManagementConfirmModal().hide();

			if (!this.hideDataManagementModal) {
				this.dataManagementModal().show();
			}
		},
		fileChanged(file: File) {
			this.file = file;
		},
		async call(action: Promise<AxiosResponse>): Promise<AxiosResponse | null> {
			let r = null;
			this.loading = true;

			try {
				const res = await action;

				if (res.status === 200 || res.status === 204) {
					if (!isLoggedIn()) {
						this.iframeHint = true;
					} else {
						if (this.confirmType !== "backup") {
							this.hideDataManagementModal = true;
						}

						r = res;
						this.closeDataManagementConfirmModal();
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
			return r;
		},
		async downloadBackup() {
			const res = await this.call(
				api.post(
					"/system/backup",
					{ password: this.password },
					{ responseType: "blob", validateStatus }
				)
			);
			if (res) {
				downloadFile(res);
			}
		},
		async restoreDatabase() {
			const formData = new FormData();
			formData.append("password", this.password);
			formData.append("file", this.file!);

			const res = await this.call(api.post("/system/restore", formData, { validateStatus }));

			if (res) {
				this.navigateHomeAfterRestart = true;
				showRestarting();
			}
		},
		async resetDatabase() {
			const res = await this.call(
				api.post(
					"/system/reset",
					{ password: this.password, ...this.selectedReset },
					{ validateStatus }
				)
			);

			if (res) {
				this.navigateHomeAfterRestart = true;
				showRestarting();
			}
		},
		async submit() {
			if (this.confirmType === "backup") {
				await this.downloadBackup();
			} else if (this.confirmType === "restore") {
				await this.restoreDatabase();
			} else {
				await this.resetDatabase();
			}
		},
		confirmModalClose() {
			if (!this.hideDataManagementModal) {
				this.dataManagementModal().show();
			}
		},
		dataManagementModalClosed() {
			if (this.confirmType === "") {
				this.reset();
			}
		},
	},
});
</script>
