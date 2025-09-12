<template>
	<div>
		<p>
			<span>{{ $t("config.general.customHelp") + " " }}</span>
			<a :href="docsLink" target="_blank">
				{{ $t("config.general.docsLink") }}
			</a>
		</p>
		<YamlEditorContainer v-model="localValue" :error-line="errorLine" />
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import YamlEditorContainer from "../YamlEditorContainer.vue";
import { docsPrefix } from "@/i18n";
import { type DeviceType } from "@/types/evcc";

export default defineComponent({
	name: "YamlEntry",
	components: {
		YamlEditorContainer,
	},
	props: {
		modelValue: String,
		type: { type: String as () => DeviceType, required: true },
		errorLine: { type: [Number, null], default: null },
	},
	emits: ["update:modelValue"],
	data() {
		return {
			localValue: this.modelValue,
		};
	},
	computed: {
		docsLink() {
			return `${docsPrefix()}/docs/devices/plugins#${this.type}`;
		},
	},
	watch: {
		modelValue: {
			handler(newVal) {
				if (this.localValue !== newVal) {
					this.localValue = newVal;
				}
			},
			immediate: true,
		},
		localValue: {
			handler(newVal) {
				this.$emit("update:modelValue", newVal);
			},
		},
	},
});
</script>
