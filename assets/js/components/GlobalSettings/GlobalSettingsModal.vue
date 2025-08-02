<template>
	<GenericModal
		id="globalSettingsModal"
		:title="$t('settings.title')"
		data-testid="global-settings-modal"
	>
		<UserInterfaceSettings :sponsor="sponsor" :telemetry="telemetry" />
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import UserInterfaceSettings from "./UserInterfaceSettings.vue";
import type { Sponsor } from "@/types/evcc";

export default defineComponent({
	name: "GlobalSettingsModal",
	components: { GenericModal, UserInterfaceSettings },
	props: {
		sponsor: { type: Object as PropType<Sponsor>, default: () => ({}) },
		telemetry: Boolean,
	},
});
</script>
