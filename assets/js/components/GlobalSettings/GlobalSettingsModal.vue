<template>
	<GenericModal
		id="globalSettingsModal"
		:title="$t('settings.title')"
		data-testid="global-settings-modal"
	>
		<UserInterfaceSettings :loadpoints="loadpoints" />
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import UserInterfaceSettings from "./UserInterfaceSettings.vue";
import type { LoadpointCompact } from "@/types/evcc";

export default defineComponent({
	name: "GlobalSettingsModal",
	components: { GenericModal, UserInterfaceSettings },
	props: {
		loadpoints: { type: Array as PropType<LoadpointCompact[]>, default: () => [] },
	},
});
</script>
