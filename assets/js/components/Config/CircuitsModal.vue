<template>
	<YamlModal
		id="circuitsModal"
		:title="$t('config.circuits.title')"
		:description="$t('config.circuits.description')"
		docs="/docs/features/loadmanagement"
		:defaultYaml="defaultYaml"
		removeKey="circuits"
		endpoint="/config/circuits"
		data-testid="circuits-modal"
		@changed="$emit('changed')"
	/>
</template>

<script>
import YamlModal from "./YamlModal.vue";
import defaultYaml from "./defaultYaml/circuits.yaml?raw";

export default {
	name: "CircuitsModal",
	components: { YamlModal },
	emits: ["changed"],
	data() {
		return { defaultYaml: defaultYaml.trim() };
	},
};
</script>
