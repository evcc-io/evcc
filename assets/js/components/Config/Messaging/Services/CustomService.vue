<template>
	<div>
		<MessagingFormRow
			:serviceType="serviceType"
			inputName="encoding"
			optional
			:helpTranslationParameter="{
				send: '`${send}`',
				format: getEncodingFormat,
			}"
		>
			<PropertyField
				id="messagingServiceCustomEncoding"
				property="encoding"
				type="Choice"
				class="me-2 w-25"
				:choice="Object.values(MESSAGING_SERVICE_CUSTOM_ENCODING)"
				:model-value="encoding"
				@update:model-value="$emit('update:encoding', $event)"
			/>
		</MessagingFormRow>

		<MessagingFormRow
			:serviceType="serviceType"
			inputName="send"
			:helpTranslationParameter="{
				url: '[docs.evcc.io](https://docs.evcc.io/en/docs/devices/plugins)',
			}"
		>
			<YamlEditorContainer
				:model-value="getContent()"
				@change="setContent($event.target.value)"
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_CUSTOM_ENCODING, MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import type { PropType } from "vue";
import PropertyField from "../../PropertyField.vue";
import YamlEditorContainer from "../../YamlEditorContainer.vue";
import MessagingFormRow from "./MessagingFormRow.vue";

const DEAFULT_SEND_PLUGIN = {
	cmd: `/usr/local/bin/evcc_message "{{.send}}"`,
	source: "script",
};

export default {
	name: "CustomService",
	components: { MessagingFormRow, PropertyField, YamlEditorContainer },
	props: {
		encoding: String as PropType<MESSAGING_SERVICE_CUSTOM_ENCODING>,
		send: Object,
	},
	emits: ["update:encoding", "update:send"],
	data() {
		return {
			serviceType: MESSAGING_SERVICE_TYPE.CUSTOM,
			MESSAGING_SERVICE_CUSTOM_ENCODING,
			DEAFULT_SEND_PLUGIN,
		};
	},
	computed: {
		getEncodingFormat() {
			switch (this.encoding) {
				case MESSAGING_SERVICE_CUSTOM_ENCODING.JSON:
					return '`{ "msg": <MSG>, "title": <TITLE> }`';
				case MESSAGING_SERVICE_CUSTOM_ENCODING.CSV:
					return "`<TITLE>,<MSG>`";
				case MESSAGING_SERVICE_CUSTOM_ENCODING.TITLE:
					return "`<TITLE>`";
				case MESSAGING_SERVICE_CUSTOM_ENCODING.TSV:
					return "`<TITLE><TAB><MSG>`";
				default:
					return "`<MSG>`";
			}
		},
	},
	mounted() {
		if (!this.send) {
			this.updateSend(DEAFULT_SEND_PLUGIN);
		}
	},
	methods: {
		updateSend(v: object) {
			this.$emit("update:send", v);
		},
		setContent(v: string) {
			this.updateSend(
				Object.fromEntries(
					v.split("\n").map((line) => {
						const [key, ...rest] = line.split(":");
						return [(key || "").trim(), rest.join(":").trim()];
					})
				)
			);
		},
		getContent() {
			return Object.entries(this.send ?? {})
				.map(([key, value]) => `${key}: ${value}`)
				.join("\n");
		},
	},
};
</script>
