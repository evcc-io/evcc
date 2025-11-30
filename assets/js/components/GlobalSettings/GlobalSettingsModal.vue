<template>
	<GenericModal
		id="globalSettingsModal"
		:title="$t('settings.title')"
		data-testid="global-settings-modal"
	>
		<UserInterfaceSettings :loadpoints="uiLoadpoints" />
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import UserInterfaceSettings from "./UserInterfaceSettings.vue";
import type { UiLoadpoint } from "@/types/evcc";

export default defineComponent({
	name: "GlobalSettingsModal",
	components: { GenericModal, UserInterfaceSettings },
	props: {
		uiLoadpoints: { type: Array as PropType<UiLoadpoint[]>, default: () => [] },
	},
});
</script>
