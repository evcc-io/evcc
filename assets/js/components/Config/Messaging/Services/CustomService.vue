<template>
	<div>
		<MessagingFormRow
			:serviceType="serviceType"
			inputName="encoding"
			optional
			:helpI18nParams="{
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
			:helpI18nParams="{
				url: '[docs.evcc.io](https://docs.evcc.io/en/docs/devices/plugins)',
			}"
		>
			<YamlEditorContainer
				:model-value="getContent"
				@update:model-value="storeNewContent($event)"
			/>
			<button
				v-if="changed"
				type="button"
				class="d-flex btn btn-sm btn-outline-primary border-0 align-items-center gap-2 ms-auto px-0 text-decoration-underline"
				tabindex="0"
				@click="formatAndSave"
			>
				{{ $t("config.messaging.service.custom.formatAndSave") }}
			</button>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_CUSTOM_ENCODING, MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import type { PropType } from "vue";
import PropertyField from "../../PropertyField.vue";
import YamlEditorContainer from "../../YamlEditorContainer.vue";
import MessagingFormRow from "./MessagingFormRow.vue";
import { load, dump } from "js-yaml";

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
			newContent: {},
			changed: false,
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
		getContent() {
			return dump(this.send);
		},
	},
	mounted() {
		if (!this.send || Object.entries(this.send).length === 0) {
			this.updateSend(DEAFULT_SEND_PLUGIN);
		}
	},
	methods: {
		updateSend(v: object) {
			this.$emit("update:send", v);
		},
		storeNewContent(s: string) {
			try {
				const o = load(s);
				if (o && typeof o === "object") {
					this.newContent = o;
					this.changed = true;
				}
			} catch {
				// tslint:disable-line:no-empty
			}
		},
		formatAndSave() {
			this.updateSend(this.newContent);
			this.changed = false;
		},
	},
};
</script>
