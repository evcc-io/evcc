<template>
	<FormRow :id="id" :label="label" :help="help" :example="example" :optional="optional">
		<slot />
	</FormRow>
</template>

<script lang="ts">
import { MESSAGING_SERVICE_TYPE } from "@/types/evcc";
import type { PropType } from "vue";
import FormRow from "../../FormRow.vue";
import formatter from "@/mixins/formatter";

export default {
	name: "MessagingFormRow",
	components: { FormRow },
	mixins: [formatter],
	props: {
		serviceType: {
			type: String as PropType<MESSAGING_SERVICE_TYPE>,
			required: true,
		},
		inputName: {
			type: String,
			required: true,
		},
		helpI18nParameter: {
			type: Object,
			default: () => {
				return {};
			},
		},
		example: String,
		optional: Boolean,
	},
	computed: {
		id() {
			return `messagingService${this.capitalizeFirstLetter(this.serviceType)}${this.capitalizeFirstLetter(this.inputName)}`;
		},
		i18n() {
			return `config.messaging.service.${this.serviceType}.${this.inputName}`;
		},
		label() {
			return this.$t(this.i18n);
		},
		help() {
			return this.$t(this.i18n + "Help", this.helpI18nParameter);
		},
	},
};
</script>
