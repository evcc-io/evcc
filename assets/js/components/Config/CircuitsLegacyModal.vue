<template>
	<YamlModal
		id="circuitsLegacyModal"
		name="circuitslegacy"
		:title="`${$t('config.circuits.title')} (${$t('config.general.legacy')})`"
		:description="$t('config.circuits.description')"
		docs="/docs/reference/configuration/circuits"
		:defaultYaml="defaultYaml"
		endpoint="/config/circuits"
		removeKey="circuits"
		data-testid="circuits-legacy-modal"
		@changed="$emit('changed')"
	>
		<template #afterDescription>
			<div class="alert alert-warning my-4" role="alert">
				{{ $t("config.circuits.legacyWarning") }}
			</div>
		</template>
	</YamlModal>
</template>

<script>
import YamlModal from "./YamlModal.vue";
import defaultYaml from "./defaultYaml/circuits.yaml?raw";

export default {
	name: "CircuitsLegacyModal",
	components: { YamlModal },
	emits: ["changed"],
	data() {
		return { defaultYaml: defaultYaml.trim() };
	},
};
</script>
