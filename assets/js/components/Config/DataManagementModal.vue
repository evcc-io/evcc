<template>
	<GenericModal
		id="dataManagementModal"
		:title="$t('config.system.dataManagement.title')"
		data-testid="data-management-modal"
		@opened="$emit('opened')"
	>
		<p>
			<span>{{ $t("config.system.dataManagement.description") }}</span>
		</p>

		<div>
			<h6>
				{{ $t("config.system.dataManagement.backup.title") }} <small>backup-summary</small>
			</h6>
			<button class="btn btn-outline-secondary" @click="openBackupConfirmModal">
				{{ $t("config.system.dataManagement.backup.download") }}
			</button>
		</div>
		<div>
			<h6>{{ $t("config.system.dataManagement.restore") }} <small>backup-restore</small></h6>
			<input class="form-control filestyle" type="file" id="formFile" data-input="false" data-buttonText="Your label here."/>
		</div>
		<div>
			<h6>{{ $t("config.system.dataManagement.reset") }} <small>backup-reset</small></h6>
		</div>

		<form ref="form" class="container mx-0 px-0"></form>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import api, { downloadFile } from "@/api";
import type { LoginAction } from "@/types/evcc";

export default defineComponent({
	name: "DataManagementModal",
	components: { GenericModal },
	emits: ["openBackupConfirmModal", "opened"],
	methods: {
		openBackupConfirmModal() {
			this.$emit("openBackupConfirmModal", (async (password: string) => {
				const res = await api.post(
					"/config/backup",
					{ password },
					{
						responseType: "blob",
						validateStatus: (code: number) => [200, 401, 403].includes(code),
					}
				);

				downloadFile(res);

				return { status: res.status };
			}) satisfies LoginAction);
		},
	},
});
</script>
