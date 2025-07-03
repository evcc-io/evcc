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

		<div class="mb-3">
			<h6>
				{{ $t("config.system.dataManagement.backup.title") }}
			</h6>

			<p>
				{{ $t("config.system.dataManagement.backup.description") }}
			</p>
			<button class="btn btn-outline-secondary" @click="$emit('openBackupConfirmModal')">
				{{ $t("config.system.dataManagement.backup.download") }}
			</button>
		</div>
		<div class="mb-3">
			<h6>
				{{ $t("config.system.dataManagement.restore.title") }}
			</h6>
			<p>
				{{ $t("config.system.dataManagement.restore.description") }}
			</p>

			<form @submit="restoreDatabase">
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

			<form @submit="resetDatabase">
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
								{{ $t("config.system.dataManagement.reset.sessionsDescription") }}
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
								{{ $t("config.system.dataManagement.reset.settingsDescription") }}
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
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import api from "@/api";
import PropertyFileField from "./PropertyFileField.vue";
import FormRow from "./FormRow.vue";

export default defineComponent({
	name: "DataManagementModal",
	components: { GenericModal, PropertyFileField, FormRow },
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
