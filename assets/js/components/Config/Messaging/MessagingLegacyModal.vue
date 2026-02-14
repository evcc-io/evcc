<template>
	<YamlModal
		id="messagingLegacyModal"
		name="messaginglegacy"
		:title="`${$t('config.messaging.title')} (${$t('config.general.legacy')})`"
		:description="$t('config.messaging.description')"
		docs="/docs/reference/configuration/messaging"
		:defaultYaml="defaultYaml"
		endpoint="/config/messaging"
		removeKey="messaging"
		data-testid="messaging-legacy-modal"
		@changed="$emit('changed')"
	>
		<template #afterDescription>
			<div class="alert alert-warning my-4" role="alert">
				{{ $t("config.messaging.legacyWarning") }}
			</div>
		</template>
	</YamlModal>
</template>

<script>
import YamlModal from "../YamlModal.vue";
import defaultYaml from "../defaultYaml/messaging.yaml?raw";

export default {
	name: "MessagingLegacyModal",
	components: { YamlModal },
	emits: ["changed"],
	data() {
		return { defaultYaml: defaultYaml.trim() };
	},
};
</script>
