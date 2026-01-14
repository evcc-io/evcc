<template>
	<FormRow :id="id" :label="label" :help="help" :example="example" :optional="optional">
		<slot />
	</FormRow>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import type { PropType } from "vue";
import FormRow from "../../FormRow.vue";

export default {
	name: "MessagingFormRow",
	components: { FormRow },
	props: {
		id: { type: String, required: true },
		serviceType: { type: String as PropType<MESSAGING_SERVICE_TYPE>, required: true },
		inputName: { type: String, required: true },
		helpI18nParams: { type: Object, default: () => ({}) },
		example: String,
		optional: Boolean,
	},
	computed: {
		i18n() {
			return `config.messaging.service.${this.serviceType}.${this.inputName}`;
		},
		label() {
			return this.$t(this.i18n);
		},
		help() {
			return this.$t(this.i18n + "Help", this.helpI18nParams);
		},
	},
};
</script>
