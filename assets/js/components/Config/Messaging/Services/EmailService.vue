<template>
	<div v-for="p in emailProperties" :key="p">
		<MessagingFormRow
			:serviceType="service.type"
			:inputName="p"
			:example="EMAIL_PROPERTIES_EXAMPLES[p]"
		>
			<PropertyField
				:id="'messagingServiceEmail' + p"
				:model-value="decoded[p] ?? ''"
				type="String"
				required
				@update:model-value="(e) => updateEmail(p, e)"
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceEmail } from "@/types/evcc";
import type { PropType } from "vue";
import PropertyField from "../../PropertyField.vue";
import MessagingFormRow from "./MessagingFormRow.vue";

const EMAIL_PROPERTIES = {
	HOST: "host",
	PORT: "port",
	USER: "user",
	PASSWORD: "password",
	FROM: "from",
	TO: "to",
} as const;

const EMAIL_PROPERTIES_EXAMPLES = {
	[EMAIL_PROPERTIES.HOST]: "emailserver.example.com",
	[EMAIL_PROPERTIES.PORT]: "587",
	[EMAIL_PROPERTIES.USER]: "john.doe",
	[EMAIL_PROPERTIES.PASSWORD]: "secret123",
	[EMAIL_PROPERTIES.FROM]: "john.doe@mail.com",
	[EMAIL_PROPERTIES.TO]: "recipient@mail.com",
} as const;

export default {
	name: "EmailService",
	components: { MessagingFormRow, PropertyField },
	props: {
		service: {
			type: Object as PropType<MessagingServiceEmail>,
			required: true,
		},
	},
	data() {
		return { emailProperties: Object.values(EMAIL_PROPERTIES), EMAIL_PROPERTIES_EXAMPLES };
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
