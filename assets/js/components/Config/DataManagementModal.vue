<template>
	<GenericModal
		id="dataManagementModal"
		:title="$t('config.system.dataManagement.title')"
		data-testid="data-management-modal"
		@opened="$emit('opened')"
		@closed="$emit('closed')"
	>
		<p>
			<span>{{ $t("config.system.dataManagement.description") }}</span>
		</p>

		<div>
			<h6>
				{{ $t("config.system.dataManagement.backup.title") }} <small>backup-summary</small>
			</h6>
			<button class="btn btn-outline-secondary" @click="$emit('openBackupConfirmModal')">
				{{ $t("config.system.dataManagement.backup.download") }}
			</button>
		</div>
		<div>
			<h6>
				{{ $t("config.system.dataManagement.restore.title") }} <small>backup-restore</small>
			</h6>

			<form @submit="restoreDatabase">
				<PropertyFileField
					ref="fileInput"
					:accepted="['.db']"
					required
					@file-changed="fileChanged"
				/>

				<button class="btn btn-outline-secondary mt-2" :disabled="file === null">
					{{ $t("config.system.dataManagement.restore.action") }}
				</button>
			</form>
		</div>
		<div>
			<h6>
				{{ $t("config.system.dataManagement.reset.title") }} <small>backup-reset</small>
			</h6>

			<form @submit="resetDatabase">
				<div class="d-flex flex-column">
					<div>
						<input
							id="resetSessions"
							v-model="selectedReset.sessions"
							class="form-check-input"
							type="checkbox"
						/>
						<label class="form-check-label ms-2" for="resetSessions">
							{{ $t("header.sessions") }}
						</label>
					</div>

					<div>
						<input
							id="resetSettings"
							v-model="selectedReset.settings"
							class="form-check-input"
							type="checkbox"
						/>
						<label class="form-check-label ms-2" for="resetSettings">
							{{ $t("config.system.dataManagement.reset.settings") }}
						</label>
					</div>
				</div>

				<button
					class="btn btn-outline-secondary mt-3"
					:disabled="!selectedReset.sessions && !selectedReset.settings"
				>
					{{ $t("config.system.dataManagement.reset.action") }}
				</button>
			</form>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import api from "@/api";
import PropertyFileField from "./PropertyFileField.vue";

export default defineComponent({
	name: "DataManagementModal",
	components: { GenericModal, PropertyFileField },
	emits: ["openBackupConfirmModal", "opened", "closed"],
	data() {
		return {
			selectedReset: {
				sessions: false,
				settings: false,
			},
			file: null as File | null,
		};
	},
	methods: {
		reset() {
			this.selectedReset = {
				sessions: false,
				settings: false,
			};
			this.file = null;
			(
				this.$refs["fileInput"] as InstanceType<typeof PropertyFileField> | undefined
			)?.reset();
		},

		fileChanged(file: File) {
			this.file = file;
		},
		async restoreDatabase() {
			const formData = new FormData();
			formData.append("file", this.file!);
			await api.post("/config/restore", formData, {
				headers: { "Content-Type": "multipart/form-data" },
			});
		},
		async resetDatabase() {
			await api.post("/config/reset", this.selectedReset);
		},
	},
});
</script>
