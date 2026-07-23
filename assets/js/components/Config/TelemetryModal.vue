<template>
	<GenericModal
		id="telemetryModal"
		:title="$t('config.telemetry.title')"
		size="lg"
		config-modal-name="telemetry"
		data-testid="telemetry-modal"
	>
		<p>{{ $t("config.telemetry.description") }}</p>
		<TelemetrySettings :sponsorActive="isSponsor" :telemetry="telemetry" />
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import TelemetrySettings from "../TelemetrySettings.vue";

export default defineComponent({
	name: "TelemetryModal",
	components: { GenericModal, TelemetrySettings },
	props: {
		isSponsor: { type: Boolean },
		telemetry: { type: Boolean },
	},
});
</script>
