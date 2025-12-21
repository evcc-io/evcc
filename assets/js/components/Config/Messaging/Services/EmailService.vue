<template>
	<div v-for="field in fields" :key="field.key">
		<MessagingFormRow
			:serviceType="service.type"
			:inputName="field.key"
			:example="field.example"
		>
			<PropertyField
				:id="field.id"
				:model-value="field.value"
				type="String"
				required
				@update:model-value="(e) => updateEmail(field.key, e)"
			/>
		</MessagingFormRow>
	</div>
</template>

<script lang="ts">
import { type MessagingServiceEmail } from "@/types/evcc";
import type { PropType } from "vue";
import PropertyField from "../../PropertyField.vue";
import MessagingFormRow from "./MessagingFormRow.vue";

type EmailProperties = {
	host: string;
	port: string;
	user: string;
	password: string;
	from: string;
	to: string;
};

export default {
	name: "EmailService",
	components: { MessagingFormRow, PropertyField },
	props: {
		service: {
			type: Object as PropType<MessagingServiceEmail>,
			required: true,
		},
	},
	computed: {
		decoded(): EmailProperties {
			const emailOther = this.service.other;

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
				host: hostname,
				port: port,
				user: username,
				password: password,
				from: from,
				to: to,
			};
		},
		fields() {
			const fieldExamples: EmailProperties = {
				host: "emailserver.example.com",
				port: "587",
				user: "john.doe",
				password: "secret123",
				from: "john.doe@mail.com",
				to: "recipient@mail.com",
			};

			return (Object.entries(fieldExamples) as [keyof EmailProperties, string][]).map(
				([key, example]) => ({
					key,
					id: `messagingServiceEmail${key}`,
					value: this.decoded[key] ?? "",
					example: example,
				})
			);
		},
	},
	methods: {
		updateEmail(p: string, v: string) {
			const emailOther = this.service.other;
			const d = { ...this.decoded, [p]: v };
			emailOther.uri = `smtp://${d.user}:${d.password}@${d.host}:${d.port}/?fromAddress=${d.from}&toAddresses=${d.to}`;
		},
	},
};
</script>
