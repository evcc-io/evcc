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
			<button
				class="btn btn-outline-secondary"
				@click="$emit('openConfirmWithPasswordModal')"
			>
				{{ $t("config.system.dataManagement.backup.download") }}
			</button>
		</div>
		<div>
			<h6>{{ $t("config.system.dataManagement.restore") }} <small>backup-restore</small></h6>
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
import Modal from "bootstrap/js/dist/modal";

export default defineComponent({
	name: "DataManagementModal",
	components: { GenericModal },
	emits: ["openConfirmWithPasswordModal", "opened"],
	data() {
		return {
			subModalOpened: false,
		};
	},
	methods: {
		openConfirmWithPasswordModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("confirmWithPasswordModal") as HTMLElement
			);
			modal.show();
			this.$emit("openConfirmWithPasswordModal");
		},
	},
});
</script>
