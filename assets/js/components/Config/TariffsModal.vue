<template>
	<YamlModal
		id="tariffsModal"
		:title="$t('config.tariff.title')"
		:description="$t('config.tariff.description')"
		docs="/docs/tariffs"
		:defaultYaml="defaultYaml"
		removeKey="tariffs"
		endpoint="/config/tariffs"
		data-testid="tariffs-modal"
		@changed="$emit('changed')"
	/>
</template>

<script>
import YamlModal from "./YamlModal.vue";
import defaultYaml from "./defaultYaml/tariffs.yaml?raw";

export default {
	name: "TariffsModal",
	components: { YamlModal },
	emits: ["changed"],
	data() {
		return { defaultYaml: defaultYaml.trim() };
	},
};
</script>
