<template>
	<JsonModal
		id="messagingModal"
		:title="$t('config.messaging.title')"
		:description="$t('config.messaging.description')"
		docs="/docs/reference/configuration/messaging"
		endpoint="/config/messaging"
		state-key="messaging"
		disable-remove
		data-testid="messaging-modal"
		size="xl"
		@changed="$emit('changed')"
	>
		<template #default="{ values }: { values: Messaging }">
			<div></div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import { MESSAGING_EVENTS, type Messaging } from "@/types/evcc";
import JsonModal from "./JsonModal.vue";
import defaultYaml from "./defaultYaml/messaging.yaml?raw";
import FormRow from "./FormRow.vue";

export default {
	name: "MessagingModal",
	components: { JsonModal, FormRow },
	emits: ["changed"],
	data() {
		return { defaultYaml: defaultYaml.trim(), MESSAGING_EVENTS };
	},
};
</script>
