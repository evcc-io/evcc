<template>
	<YamlModal
		id="eebusModal"
		:title="$t('config.eebus.title')"
		:description="$t('config.eebus.description')"
		docs="/docs/reference/configuration/eebus"
		:defaultYaml="defaultYaml"
		endpoint="/config/eebus"
		@changed="$emit('changed')"
	/>
</template>

<script>
import YamlModal from "./YamlModal.vue";
import defaultYaml from "./defaultYaml/eebus.yaml?raw";

export default {
	name: "EebusModal",
	components: { YamlModal },
	emits: ["changed"],
	data() {
		return { defaultYaml: defaultYaml.trim() };
	},
};
</script>
