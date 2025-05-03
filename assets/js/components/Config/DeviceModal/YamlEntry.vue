<template>
	<div>
		<p>
			<span>{{ $t("config.general.customHelp") + " " }}</span>
			<a :href="docsLink" target="_blank">
				{{ $t("config.general.docsLink") }}
			</a>
		</p>
		<YamlEditorContainer
			:modelValue="modelValue"
			@update:model-value="$emit('update:modelValue', $event)"
		/>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import YamlEditorContainer from "../YamlEditorContainer.vue";
import { docsPrefix } from "@/i18n";

type DeviceType = "vehicle" | "charger" | "meter";

export default defineComponent({
	name: "YamlEntry",
	components: {
		YamlEditorContainer,
	},
	props: {
		modelValue: String,
		type: {
			type: String as () => DeviceType,
			required: true,
		},
	},
	emits: ["update:modelValue"],
	computed: {
		docsLink() {
			return `${docsPrefix()}/docs/devices/plugins#${this.type}`;
		},
	},
});
</script>
