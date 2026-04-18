<template>
	<JsonModal
		name="ftpbackup"
		:title="$t('config.ftpbackup.title')"
		:description="$t('config.ftpbackup.description')"
		endpoint="/config/ftpbackup"
		state-key="ftpbackup"
		@changed="$emit('changed')"
	>
		<template #default="{ values }">
			<FormRow
				id="ftpBackupHost"
				:label="$t('config.ftpbackup.labelHost')"
				example="192.168.1.10"
			>
				<input id="ftpBackupHost" v-model="values.host" class="form-control" required />
			</FormRow>
			<FormRow
				id="ftpBackupPort"
				:label="$t('config.ftpbackup.labelPort')"
				example="21"
				optional
			>
				<input
					id="ftpBackupPort"
					v-model.number="values.port"
					class="form-control"
					type="number"
					min="1"
					max="65535"
				/>
			</FormRow>
			<FormRow
				id="ftpBackupUser"
				:label="$t('config.ftpbackup.labelUser')"
				example="evcc"
				optional
			>
				<input
					id="ftpBackupUser"
					v-model="values.user"
					class="form-control"
					autocomplete="username"
				/>
			</FormRow>
			<FormRow id="ftpBackupPassword" :label="$t('config.ftpbackup.labelPassword')" optional>
				<input
					id="ftpBackupPassword"
					v-model="values.password"
					class="form-control"
					type="password"
					autocomplete="new-password"
				/>
			</FormRow>
			<FormRow
				id="ftpBackupDirectory"
				:label="$t('config.ftpbackup.labelDirectory')"
				example="/evcc/backups"
				optional
			>
				<input id="ftpBackupDirectory" v-model="values.directory" class="form-control" />
			</FormRow>
			<FormRow
				id="ftpBackupSchedule"
				:label="$t('config.ftpbackup.labelSchedule')"
				example="03:00"
				optional
			>
				<input
					id="ftpBackupSchedule"
					v-model="values.schedule"
					class="form-control"
					pattern="^([01]\\d|2[0-3]):([0-5]\\d)$"
					placeholder="03:00"
				/>
			</FormRow>
			<FormRow
				id="ftpBackupTimeout"
				:label="$t('config.ftpbackup.labelTimeout')"
				example="30s"
				optional
			>
				<input
					id="ftpBackupTimeout"
					v-model="values.timeout"
					class="form-control"
					placeholder="30s"
				/>
			</FormRow>
			<FormRow id="ftpBackupTls" :label="$t('config.ftpbackup.labelTls')">
				<div class="d-flex">
					<input
						id="ftpBackupTls"
						v-model="values.tls"
						class="form-check-input"
						type="checkbox"
					/>
					<label class="form-check-label ms-2" for="ftpBackupTls">
						{{ $t("config.ftpbackup.labelCheckTls") }}
					</label>
				</div>
			</FormRow>
			<FormRow id="ftpBackupInsecure" :label="$t('config.ftpbackup.labelInsecureSkipVerify')">
				<div class="d-flex">
					<input
						id="ftpBackupInsecure"
						v-model="values.insecureSkipVerify"
						class="form-check-input"
						type="checkbox"
					/>
					<label class="form-check-label ms-2" for="ftpBackupInsecure">
						{{ $t("config.ftpbackup.labelCheckInsecureSkipVerify") }}
					</label>
				</div>
			</FormRow>
		</template>
	</JsonModal>
</template>

<script>
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";

export default {
	name: "FtpBackupModal",
	components: { FormRow, JsonModal },
	emits: ["changed"],
};
</script>
