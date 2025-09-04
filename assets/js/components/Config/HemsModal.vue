<template>
	<YamlModal
		id="hemsModal"
		:title="$t('config.hems.title')"
		:description="$t('config.hems.description')"
		docs="/docs/features/14a-enwg-steuve"
		:defaultYaml="defaultYaml"
		endpoint="/config/hems"
		removeKey="hems"
		data-testid="hems-modal"
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
