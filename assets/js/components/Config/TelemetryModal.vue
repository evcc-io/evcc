<template>
	<GenericModal
		id="telemetryModal"
		:title="$t('config.telemetry.title')"
		size="lg"
		data-testid="telemetry-modal"
	>
		<p>{{ $t("config.telemetry.description") }}</p>
		<TelemetrySettings :sponsorActive="sponsorActive" :telemetry="telemetry" />
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import TelemetrySettings from "../TelemetrySettings.vue";
import type { Sponsor } from "../../types/evcc";

export default defineComponent({
	name: "TelemetryModal",
	components: { GenericModal, TelemetrySettings },
	props: {
		sponsor: { type: Object as PropType<Sponsor> },
		telemetry: { type: Boolean },
	},
	computed: {
		sponsorActive(): boolean {
			return !!this.sponsor?.status.name;
		},
	},
});
</script>
