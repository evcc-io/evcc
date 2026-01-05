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
	>
		<template #extra>
			<p class="my-2 small">
				{{ $t("config.circuits.usableMeters") }}:
				<code v-for="meter in usableMeters" :key="meter.name" class="ms-1 meter">
					{{ meter.name }}<span v-if="meter.title" class="ms-1">({{ meter.title }})</span>
				</code>
			</p>
		</template>
	</YamlModal>
</template>

<script lang="ts">
import YamlModal from "./YamlModal.vue";
import defaultYaml from "./defaultYaml/circuits.yaml?raw";
import type { ConfigMeter } from "@/types/evcc";
import type { PropType } from "vue";

export default {
	name: "CircuitsModal",
	components: { YamlModal },
	props: {
		gridMeter: { type: Object as PropType<ConfigMeter>, default: null },
		extMeters: { type: Array as PropType<ConfigMeter[]>, default: () => [] },
	},
	emits: ["changed"],
	data() {
		return { defaultYaml: defaultYaml.trim() };
	},
	computed: {
		usableMeters() {
			const result = [];
			if (this.gridMeter) {
				result.push({ name: this.gridMeter.name, title: this.$t("config.grid.title") });
			}
			if (this.extMeters) {
				result.push(
					...this.extMeters.map((m) => ({
						name: m.name,
						title: m.deviceTitle || m.deviceProduct || m.config["template"] || m.type,
					}))
				);
			}
			return result;
		},
	},
};
</script>
<style scoped>
.meter:not(:last-child)::after {
	content: ",";
}
</style>
