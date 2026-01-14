<template>
	<div>
		<MessagingFormRow
			:id="formId('encoding')"
			:serviceType="serviceType"
			inputName="encoding"
			optional
			:helpI18nParams="{
				send: '`{{.send}}`',
				format: encodingFormat,
			}"
		>
			<PropertyField
				:id="formId('encoding')"
				property="encoding"
				type="Choice"
				class="me-2 w-25"
				:choice="encodingChoices"
				:model-value="encoding"
				@update:model-value="$emit('update:encoding', $event)"
			/>
		</MessagingFormRow>

		<MessagingFormRow
			:id="formId('send')"
			:serviceType="serviceType"
			inputName="send"
			:helpI18nParams="{
				url: '[docs.evcc.io](https://docs.evcc.io/en/docs/devices/plugins)',
			}"
		>
			<YamlEditorContainer :model-value="send" @update:model-value="updateSend" />
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_CUSTOM_ENCODING, MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import type { PropType } from "vue";
import PropertyField from "../../PropertyField.vue";
import YamlEditorContainer from "../../YamlEditorContainer.vue";
import MessagingFormRow from "./MessagingFormRow.vue";
import { formId } from "../utils";

const DEAFULT_SEND_PLUGIN = `source: script
cmd: /usr/local/bin/evcc_message "{{.send}}"
`;

const FORMATS: Record<MESSAGING_SERVICE_CUSTOM_ENCODING, string> = {
	json: '`{ "msg": <MSG>, "title": <TITLE> }`',
	csv: "`<TITLE>,<MSG>`",
	title: "`<TITLE>`",
	tsv: "`<TITLE><TAB><MSG>`",
};

export default {
	name: "CustomService",
	components: { MessagingFormRow, PropertyField, YamlEditorContainer },
	props: {
		encoding: String as PropType<MESSAGING_SERVICE_CUSTOM_ENCODING>,
		send: String,
	},
	emits: ["update:encoding", "update:send"],
	data() {
		return {
			newContent: {},
			changed: false,
		};
	},
	computed: {
		encodingChoices() {
			return Object.values(MESSAGING_SERVICE_CUSTOM_ENCODING);
		},
		encodingFormat() {
			return this.encoding ? FORMATS[this.encoding] : "`<MSG>`";
		},
		serviceType() {
			return MESSAGING_SERVICE_TYPE.CUSTOM;
		},
	},
	mounted() {
		if (!this.send) {
			this.updateSend(DEAFULT_SEND_PLUGIN);
		}
	},
	methods: {
		formId(name: string) {
			return formId(this.serviceType, name);
		},
		updateSend(v: string) {
			this.$emit("update:send", v);
		},
	},
};
</script>
