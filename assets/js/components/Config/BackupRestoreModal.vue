<template>
	<div>
		<GenericModal
			id="backupRestoreModal"
			:title="$t('config.system.backupRestore.title')"
			data-testid="backup-restore-modal"
			@closed="backupRestoreModalClosed"
		>
			<p>
				<span>{{ $t("config.system.backupRestore.description") }}</span>
			</p>

			<div class="mb-3">
				<h6>
					{{ $t("config.system.backupRestore.backup.title") }}
				</h6>

				<p>
					{{ $t("config.system.backupRestore.backup.description") }}
				</p>
				<button
					class="btn btn-outline-secondary"
					@click="openBackupRestoreConfirmModal('backup')"
				>
					{{ $t("config.system.backupRestore.backup.action") }}
				</button>
			</div>
			<div class="mb-3">
				<h6>
					{{ $t("config.system.backupRestore.restore.title") }}
				</h6>
				<p>
					{{ $t("config.system.backupRestore.restore.description") }}
				</p>

				<form @submit.prevent="openBackupRestoreConfirmModal('restore')">
					<FormRow
						id="restoreFile"
						:label="$t('config.system.backupRestore.restore.labelFile')"
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
						{{ $t("config.system.backupRestore.restore.action") }}
					</button>
				</form>
			</div>
			<div class="mb-3">
				<h6>
					{{ $t("config.system.backupRestore.reset.title") }}
				</h6>
				<p>{{ $t("config.system.backupRestore.reset.description") }}</p>

				<form @submit.prevent="openBackupRestoreConfirmModal('reset')">
					<div class="d-flex flex-column mb-2">
						<div class="d-flex mb-1">
							<input
								id="resetSessions"
								v-model="selectedReset.sessions"
								class="form-check-input"
								type="checkbox"
							/>
							<label class="form-check-label ms-2" for="resetSessions">
								<span>{{ $t("config.system.backupRestore.reset.sessions") }}</span>
								<br />
								<small>
									{{
										$t("config.system.backupRestore.reset.sessionsDescription")
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
								<span>{{ $t("config.system.backupRestore.reset.settings") }}</span>
								<br />
								<small>
									{{
										$t("config.system.backupRestore.reset.settingsDescription")
									}}
								</small>
							</label>
						</div>
					</div>

					<button
						class="btn btn-outline-danger mt-3"
						:disabled="!selectedReset.sessions && !selectedReset.settings"
					>
						{{ $t("config.system.backupRestore.reset.action") }}
					</button>
				</form>
			</div>
			<p>
				<small>
					{{ $t("config.system.backupRestore.note") }}
				</small>
			</p>
		</GenericModal>
		<GenericModal
			id="backupRestoreConfirmModal"
			:title="$t(`config.system.backupRestore.${confirmType}.title`)"
			size="md"
			data-testid="backup-restore-confirm-modal"
			@close="confirmModalClose"
			@closed="confirmType = ''"
		>
			<form @submit.prevent="submit">
				<p>
					<span>{{
						$t(`config.system.backupRestore.${confirmType}.confirmationText`)
					}}</span>
				</p>

				<PasswordInput
					v-if="!authDisabled"
					v-model:password="password"
					:error="error"
					:iframe-hint="iframeHint"
				/>

				<div class="d-flex justify-content-between gap-2 flex-wrap">
					<button
						:disabled="loading"
						type="button"
						class="btn btn-outline-secondary"
						data-bs-dismiss="modal"
					>
						<span>{{ $t(`config.system.backupRestore.cancel`) }}</span>
					</button>

					<button
						type="submit"
						class="btn text-truncate"
						:class="confirmType === 'backup' ? 'btn-primary' : 'btn-danger'"
						:disabled="loading"
					>
						<span
							v-if="loading"
							class="spinner-border spinner-border-sm"
							role="status"
							aria-hidden="true"
						></span>
						{{ $t(`config.system.backupRestore.${confirmType}.confirmationButton`) }}
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
	name: "BackupRestoreModal",
	components: { GenericModal, PropertyFileField, FormRow, PasswordInput },
	props: {
		authDisabled: Boolean,
	},
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
			hideBackupRestoreModal: false,
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
		resetBackupRestoreConfirmModal() {
			this.password = "";
			this.loading = false;
			this.iframeHint = false;
			this.error = "";
		},
		resetBackupRestoreModal() {
			this.selectedReset = {
				sessions: false,
				settings: false,
			};
			this.file = null;
			this.navigateHomeAfterRestart = false;
			this.hideBackupRestoreModal = false;
			(
				this.$refs["fileInput"] as InstanceType<typeof PropertyFileField> | undefined
			)?.reset();
		},
		reset() {
			this.resetBackupRestoreConfirmModal();
			this.resetBackupRestoreModal();
			this.backupRestoreConfirmModal().hide();
			this.backupRestoreModal().hide();
		},
		backupRestoreModal() {
			return Modal.getOrCreateInstance(
				document.getElementById("backupRestoreModal") as HTMLElement
			);
		},
		backupRestoreConfirmModal() {
			return Modal.getOrCreateInstance(
				document.getElementById("backupRestoreConfirmModal") as HTMLElement
			);
		},
		openBackupRestoreConfirmModal(type: typeof this.confirmType) {
			this.resetBackupRestoreConfirmModal();
			this.backupRestoreConfirmModal().show();
			this.backupRestoreModal().hide();
			this.confirmType = type;
		},
		closeModal() {
			this.backupRestoreModal().hide();
		},
		showModal() {
			this.backupRestoreModal().show();
		},
		closeConfirmModal() {
			this.backupRestoreConfirmModal().hide();
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
						r = res;
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
				this.closeConfirmModal();
				downloadFile(res);
			}
		},
		async restoreDatabase() {
			const formData = new FormData();
			formData.append("password", this.password);
			formData.append("file", this.file!);

			const res = await this.call(api.post("/system/restore", formData, { validateStatus }));

			if (res) {
				this.hideBackupRestoreModal = true;
				this.navigateHomeAfterRestart = true;
				this.closeConfirmModal();
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
				this.hideBackupRestoreModal = true;
				this.navigateHomeAfterRestart = true;
				this.closeConfirmModal();
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
			if (!this.hideBackupRestoreModal) {
				this.showModal();
			}
		},
		backupRestoreModalClosed() {
			if (this.confirmType === "") {
				this.reset();
			}
		},
	},
});
</script>
