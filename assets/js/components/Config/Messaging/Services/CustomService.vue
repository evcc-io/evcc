<template>
	<div>
		<MessagingFormRow
			:serviceType="service.type"
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
				:model-value="serviceData.other.encoding"
				@update:model-value="(e) => (serviceData.other.encoding = e)"
			/>
		</MessagingFormRow>

		<MessagingFormRow
			:serviceType="service.type"
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
import { type MessagingServiceCustom, MESSAGING_SERVICE_CUSTOM_ENCODING } from "@/types/evcc";
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
		service: {
			type: Object as PropType<MessagingServiceCustom>,
			required: true,
		},
	},
	data() {
		return {
			serviceData: this.service,
			MESSAGING_SERVICE_CUSTOM_ENCODING,
			DEAFULT_SEND_PLUGIN,
		};
	},
	computed: {
		getEncodingFormat() {
			switch (this.serviceData.other.encoding) {
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
		if (!this.serviceData.other.send) {
			this.serviceData.other.send = DEAFULT_SEND_PLUGIN;
		}
	},
	methods: {
		setContent(v: string) {
			this.serviceData.other.send = Object.fromEntries(
				v.split("\n").map((line) => {
					const [key, ...rest] = line.split(":");
					return [(key || "").trim(), rest.join(":").trim()];
				})
			);
		},
		getContent() {
			return Object.entries(this.service.other.send)
				.map(([key, value]) => `${key}: ${value}`)
				.join("\n");
		},
	},
};
</script>
