<template>
	<YamlModal
		name="tariffsLegacy"
		:title="`${$t('config.tariff.title')} (${$t('config.general.legacy')})`"
		:description="$t('config.tariff.description')"
		docs="/docs/tariffs"
		:defaultYaml="defaultYaml"
		removeKey="tariffs"
		endpoint="/config/tariffs"
		data-testid="tariffs-legacy-modal"
		@changed="$emit('changed')"
	>
		<template #afterDescription>
			<div class="alert alert-warning my-4" role="alert">
				{{ $t("config.tariff.legacyWarning") }}
			</div>
		</template>
	</YamlModal>
</template>

<script>
import YamlModal from "./YamlModal.vue";
import defaultYaml from "./defaultYaml/tariffs.yaml?raw";

export default {
	name: "TariffsLegacyModal",
	components: { YamlModal },
	emits: ["changed"],
	data() {
		return { defaultYaml: defaultYaml.trim() };
	},
};
</script>
