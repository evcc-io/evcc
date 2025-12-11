<template>
	<div v-for="p in emailProperties" :key="p">
		<FormRow
			:id="'messagingServiceEmail' + p"
			:label="p"
			:help="$t('config.messaging.email.' + p.toLowerCase())"
		>
			<PropertyField
				:id="'messagingServiceEmail' + p"
				:model-value="decoded[p] ?? ''"
				type="String"
				required
				@update:model-value="(e) => updateEmail(p, e)"
			/>
		</FormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceEmail } from "@/types/evcc";
import FormRow from "../FormRow.vue";
import PropertyField from "../PropertyField.vue";
import type { PropType } from "vue";

const EMAIL_PROPERTIES = {
	HOST: "Host",
	PORT: "Port",
	USER: "User",
	PASSWORD: "Password",
	FROM: "From",
	TO: "To",
} as const;

export default {
	name: "EmailService",
	components: { FormRow, PropertyField },
	props: {
		service: {
			type: Object as PropType<MessagingServiceEmail>,
			required: true,
		},
	},
	data() {
		return { emailProperties: Object.values(EMAIL_PROPERTIES) };
	},
	computed: {
		decoded(): Record<string, string> {
			const emailOther = this.service.other as any;
			let hostname = "";
			let port = "";
			let username = "";
			let password = "";
			let from = "";
			let to = "";

			try {
				const url = new URL(emailOther.uri.replace(/^smtp/, "http"));
				const params = new URLSearchParams(url.search);

				hostname = url.hostname;
				port = url.port;
				username = url.username;
				password = url.password;
				from = params.get("fromAddress") ?? from;
				to = params.get("toAddresses") ?? to;
			} catch (e) {
				console.warn(e);
			}

			return {
				[EMAIL_PROPERTIES.HOST]: hostname,
				[EMAIL_PROPERTIES.PORT]: port,
				[EMAIL_PROPERTIES.USER]: username,
				[EMAIL_PROPERTIES.PASSWORD]: password,
				[EMAIL_PROPERTIES.FROM]: from,
				[EMAIL_PROPERTIES.TO]: to,
			};
		},
	},
	methods: {
		updateEmail(p: string, v: string) {
			const emailOther = this.service.other as any;
			const d = this.decoded;
			d[p] = v;
			emailOther.uri = `smtp://${d[EMAIL_PROPERTIES.USER]}:${d[EMAIL_PROPERTIES.PASSWORD]}@${d[EMAIL_PROPERTIES.HOST]}:${d[EMAIL_PROPERTIES.PORT]}/?fromAddress=${d[EMAIL_PROPERTIES.FROM]}&toAddresses=${d[EMAIL_PROPERTIES.TO]}`;
		},
	},
};
</script>
