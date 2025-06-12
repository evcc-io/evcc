<template>
	<GenericModal
		id="backupRecoveryModal"
		:title="$t('config.system.backupAndRecovery')"
		data-testid="backup-restore-modal"
	>
		<form ref="form" class="container mx-0 px-0">
			<p>Backup</p>
			<p>Restore</p>
			<p>Reset</p>
		</form>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";

export default defineComponent({
	name: "BackupRecoveryModal",
	components: { GenericModal },
	data() {
		return {
			isModalVisible: null as boolean | null,
		};
	},
	methods: {
		open() {
			this.isModalVisible = true;
		},
		closed() {
			this.isModalVisible = false;
		},
	},
});
</script>
