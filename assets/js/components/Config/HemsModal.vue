<template>
	<YamlModal
		id="hemsModal"
		:title="$t('config.hems.title')"
		:description="$t('config.hems.description')"
		docs="/docs/reference/configuration/hems"
		:defaultYaml="defaultYaml"
		endpoint="/config/hems"
		size="md"
		@changed="$emit('changed')"
	/>
</template>

<script>
import YamlModal from "./YamlModal.vue";
import defaultYaml from "./defaultYaml/hems.yaml?raw";

export default {
	name: "HemsModal",
	components: { YamlModal },
	emits: ["changed"],
	data() {
		return { defaultYaml: defaultYaml.trim() };
	},
};
</script>
