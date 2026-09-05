<template>
	<YamlModal
		id="circuitsLegacyModal"
		name="circuitslegacy"
		:title="`${$t('config.circuits.title')} (${$t('config.general.legacy')})`"
		:description="$t('config.circuits.description')"
		docs="/reference/configuration/circuits"
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
import type { ConfigMeter } from "@/types/evcc.ts";
import type { PropType } from "vue";
import YamlModal from "./YamlModal.vue";
import defaultYaml from "./defaultYaml/circuits.yaml?raw";

export default {
	name: "CircuitsLegacyModal",
	components: { YamlModal },
	props: {
		gridMeter: { type: Object as PropType<ConfigMeter>, default: null },
		extMeters: {
			type: Array as PropType<ConfigMeter[]>,
			default: () => [],
		},
	},
	emits: ["changed"],
	data() {
		return { defaultYaml: defaultYaml.trim() };
	},
	computed: {
		usableMeters() {
			const result = [];
			if (this.gridMeter) {
				result.push({
					name: this.gridMeter.name,
					title: this.$t("config.grid.title"),
				});
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
