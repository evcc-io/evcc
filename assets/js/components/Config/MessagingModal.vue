<template>
	<JsonModal
		id="messagingModal"
		:title="$t('config.messaging.title')"
		:description="$t('config.messaging.description')"
		docs="/docs/reference/configuration/messaging"
		endpoint="/config/messaging"
		state-key="messaging"
		disable-remove
		data-testid="messaging-modal"
		size="xl"
		@changed="$emit('changed')"
	>
		<template #default="{ values }: { values: Messaging }">
			<ul class="nav nav-tabs mb-4">
				<li class="nav-item">
					<a
						class="nav-link"
						:class="{ active: !activeServiceTab }"
						href="#"
						@click.prevent="activeServiceTab = null"
					>
						Events
					</a>
				</li>
				<li v-for="n in Object.values(MESSAGING_SERVICE_TYPE)" class="nav-item">
					<a
						class="nav-link text-capitalize"
						:class="{ active: activeServiceTab === n }"
						href="#"
						@click.prevent="activeServiceTab = n"
					>
						{{ n }}
					</a>
				</li>
			</ul>

			<div v-if="!activeServiceTab">
				<div v-for="event in Object.values(MESSAGING_EVENTS)" :key="event">
					<h6>Event #{{ event }}</h6>
					<FormRow :id="'messagingEventTitle' + event" label="Title">
						<PropertyField
							:id="'messagingEventTitle' + event"
							:model-value="values.events?.[event].title"
							type="String"
							required
						/>
					</FormRow>
					<FormRow :id="'messagingEventMessage' + event" label="Message">
						<PropertyField
							:id="'messagingEventMessage' + event"
							:model-value="values.events?.[event].msg"
							type="String"
							required
						/>
					</FormRow>
				</div>
			</div>
			<div v-else>
				<div
					v-for="(s, index) in values.services?.filter(
						(s) => s.type === activeServiceTab
					)"
					:key="index"
					class="mb-5"
				>
					<div class="border rounded px-3 pt-4 pb-3 mb-3">
						<div class="d-lg-block">
							<h5 class="box-heading">
								<div class="inner">Messaging #{{ index + 1 }}</div>
							</h5>
						</div>

						<div v-if="s.type === MESSAGING_SERVICE_TYPE.PUSHOVER">
							<FormRow
								id="messagingServicePushoverApp"
								label="App"
								:help="$t('config.messaging.pushover.app')"
							>
								<PropertyField
									:id="'messagingServicePushoverApp'"
									v-model="s.other.app"
									type="String"
									required
								/>
							</FormRow>
							<FormRow
								id="messagingServicePushoverRecipients"
								label="Recipients"
								:help="$t('config.messaging.pushover.recipients')"
							>
								<PropertyField
									id="messagingServicePushoverRecipients"
									v-model="s.other.devices"
									property="recipients"
									type="List"
									size="w-100"
									class="me-2"
									required
								/>
							</FormRow>
							<FormRow
								id="messagingServicePushoverDevices"
								label="Devices"
								:help="$t('config.messaging.pushover.devices')"
							>
								<PropertyField
									id="messagingServicePushoverDevices"
									v-model="s.other.devices"
									property="devices"
									type="List"
									required
								/>
							</FormRow>
						</div>
						<div v-else-if="s.type === MESSAGING_SERVICE_TYPE.TELEGRAM">
							<FormRow
								id="messagingServiceTelegramToken"
								label="Token"
								:help="$t('config.messaging.telegram.token')"
							>
								<PropertyField
									id="messagingServiceTelegramToken"
									v-model="s.other.token"
									type="String"
									required
								/>
							</FormRow>
							<FormRow
								id="messagingServiceTelegramChats"
								label="Recipients"
								:help="$t('config.messaging.telegram.chats')"
							>
								<PropertyField
									id="messagingServiceTelegramChats"
									v-model="s.other.chats"
									property="chats"
									type="List"
									required
								/>
							</FormRow>
						</div>
						<div v-else-if="s.type === MESSAGING_SERVICE_TYPE.EMAIL">
							<FormRow
								v-for="p in EMAIL_PROPERTY"
								:id="'messagingServiceEmail' + p"
								:label="p"
								:help="$t('config.messaging.email.' + p.toLowerCase())"
							>
								<PropertyField
									:id="'messagingServiceEmail' + p"
									:model-value="decodeEmail(s.other.uri)[p] ?? ''"
									@update:model-value="
										(e) => (s.other.uri = changeEmailValue(s.other.uri, p, e))
									"
									type="String"
									required
								/>
							</FormRow>
						</div>
						<div v-else-if="s.type === MESSAGING_SERVICE_TYPE.SHOUT">
							<FormRow
								id="messagingServiceShoutUri"
								label="Uri"
								:help="$t('config.messaging.shout.uri')"
							>
								<PropertyField
									id="messagingServiceShoutUri"
									v-model="s.other.uri"
									type="String"
									required
								/>
							</FormRow>
						</div>
						<div v-else-if="s.type === MESSAGING_SERVICE_TYPE.NTFY">
							<FormRow
								id="messagingServiceNftyHost"
								label="Host"
								:help="$t('config.messaging.nfty.host')"
							>
								<PropertyField
									id="messagingServiceNftyHost"
									:model-value="decodeNfty(s.other.uri)[NFTY_PROPERTY.HOST]"
									@update:model-value="
										(e) =>
											(s.other.uri = changeNftyValue(
												s.other.uri,
												NFTY_PROPERTY.HOST,
												e
											))
									"
									type="String"
									required
								/>
							</FormRow>
							<FormRow
								id="messagingServiceNftyTopics"
								label="Topics"
								:help="$t('config.messaging.nfty.topics')"
							>
								<PropertyField
									id="messagingServiceNftyTopics"
									:model-value="
										decodeNfty(s.other.uri)[NFTY_PROPERTY.TOPICS].split(',')
									"
									@update:model-value="
										(e) =>
											(s.other.uri = changeNftyValue(
												s.other.uri,
												NFTY_PROPERTY.TOPICS,
												e
											))
									"
									property="topics"
									type="List"
									required
								/>
							</FormRow>
							<FormRow
								id="messagingServiceNftyPriority"
								label="Priority"
								:help="$t('config.messaging.nfty.priority')"
								optional
							>
								<PropertyField
									id="messagingServiceNftyPriority"
									property="priority"
									type="Choice"
									class="me-2 w-25"
									:choice="Object.values(MESSAGING_SERVICE_NFTY_PRIORITY)"
									:model-value="s.other.priority"
									@update:model-value="(e) => (s.other.priority = e)"
								/>
							</FormRow>
							<FormRow
								id="messagingServiceNftyTags"
								label="Recipients"
								:help="$t('config.messaging.nfty.tags')"
								optional
							>
								<PropertyField
									id="messagingServiceNftyTags"
									:model-value="s.other.tags?.split(',')"
									@update:model-value="(e: string[]) => (s.other.tags = e.join())"
									property="tags"
									type="List"
								/>
							</FormRow>
						</div>
						<div v-else-if="s.type === MESSAGING_SERVICE_TYPE.CUSTOM">
							<FormRow
								id="messagingServiceCustomEncoding"
								label="Encoding"
								:help="$t('config.messaging.custom.encoding')"
								optional
							>
								<PropertyField
									id="messagingServiceCustomEncoding"
									property="encoding"
									type="Choice"
									class="me-2 w-25"
									:choice="Object.values(MESSAGING_SERVICE_NFTY_PRIORITY)"
									:model-value="s.other.encoding"
									@update:model-value="(e) => (s.other.encoding = e)"
								/>
							</FormRow>
							<YamlEntry type="messaging" v-model="s.other.send"></YamlEntry>
						</div>
					</div>

					<button
						type="button"
						class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray ms-auto"
						:aria-label="$t('config.general.remove')"
						tabindex="0"
						@click="
							values.services?.splice(
								values.services.findIndex((f) => deepEqual(s, f)),
								1
							)
						"
					>
						<shopicon-regular-trash
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-trash>
						{{ $t("config.general.remove") }}
					</button>
				</div>
				<hr
					v-if="values.services?.filter((s) => s.type === activeServiceTab).length != 0"
					class="mb-5"
				/>
				<button
					type="button"
					class="d-flex btn btn-sm align-items-center gap-2 mb-5"
					:class="
						values.services?.length === 0
							? 'btn-secondary mt-5'
							: 'btn-outline-secondary border-0 evcc-gray'
					"
					data-testid="networkconnection-add"
					tabindex="0"
					@click="addMessaging(values.services)"
				>
					<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
					Add messaging
				</button>
			</div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import {
	MESSAGING_EVENTS,
	MESSAGING_SERVICE_TYPE,
	MESSAGING_SERVICE_NFTY_PRIORITY,
	type Messaging,
	type MessagingServices,
	MESSAGING_SERVICE_CUSTOM_ENCODING,
} from "@/types/evcc";
import JsonModal from "./JsonModal.vue";
import defaultYaml from "./defaultYaml/messaging.yaml?raw";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import YamlEntry from "./DeviceModal/YamlEntry.vue";
import "@h2d2/shopicons/es/regular/plus";
import "@h2d2/shopicons/es/regular/trash";
import deepEqual from "@/utils/deepEqual";

enum EMAIL_PROPERTY {
	HOST = "Host",
	PORT = "Port",
	USER = "User",
	PASSWORD = "Password",
	FROM = "From",
	TO = "To",
}

enum NFTY_PROPERTY {
	HOST = "host",
	TOPICS = "topics",
}

export default {
	name: "MessagingModal",
	components: { JsonModal, FormRow, PropertyField, YamlEntry },
	emits: ["changed"],
	data() {
		return {
			defaultYaml: defaultYaml.trim(),
			MESSAGING_EVENTS,
			MESSAGING_SERVICE_TYPE,
			EMAIL_PROPERTY,
			MESSAGING_SERVICE_NFTY_PRIORITY,
			NFTY_PROPERTY,
			activeServiceTab: null as MESSAGING_SERVICE_TYPE | null,
			deepEqual,
		};
	},
	methods: {
		decodeEmail(uri: string) {
			var hostname = "";
			var port = "";
			var username = "";
			var password = "";
			var from = "";
			var to = "";

			try {
				const url = new URL(uri.replace(/^smtp/, "http"));
				const params = new URLSearchParams(url.search);

				hostname = url.hostname;
				port = url.port;
				username = url.username;
				password = url.password;
				from = params.get("fromAddress") ?? from;
				to = params.get("toAddresses") ?? to;
			} catch (error) {}

			return {
				[EMAIL_PROPERTY.HOST]: hostname,
				[EMAIL_PROPERTY.PORT]: port,
				[EMAIL_PROPERTY.USER]: username,
				[EMAIL_PROPERTY.PASSWORD]: password,
				[EMAIL_PROPERTY.FROM]: from,
				[EMAIL_PROPERTY.TO]: to,
			};
		},
		changeEmailValue(uri: string, p: EMAIL_PROPERTY, v: string) {
			var d = this.decodeEmail(uri);
			d[p] = v;
			return `smtp://${d[EMAIL_PROPERTY.USER]}:${d[EMAIL_PROPERTY.PASSWORD]}@${d[EMAIL_PROPERTY.HOST]}:${d[EMAIL_PROPERTY.PORT]}/?fromAddress=${d[EMAIL_PROPERTY.FROM]}&toAddresses=${d[EMAIL_PROPERTY.TO]}`;
		},
		decodeNfty(uri: string) {
			var hostname = "";
			var pathname = "";

			try {
				const url = new URL(uri);
				hostname = url.hostname;
				pathname = url.pathname.replace("/", "");
			} catch (e) {}

			return {
				[NFTY_PROPERTY.HOST]: hostname,
				[NFTY_PROPERTY.TOPICS]: pathname,
			};
		},
		changeNftyValue(uri: string, p: NFTY_PROPERTY, v: string) {
			var d = this.decodeNfty(uri);
			d[p] = v;
			return `https://${d[NFTY_PROPERTY.HOST]}/${d[NFTY_PROPERTY.TOPICS]}`;
		},
		addMessaging(values?: MessagingServices[]) {
			if (!values) {
				return;
			}

			var s = {} as MessagingServices;

			switch (this.activeServiceTab) {
				case MESSAGING_SERVICE_TYPE.PUSHOVER:
					s = {
						type: MESSAGING_SERVICE_TYPE.PUSHOVER,
						other: { app: "", devices: [], recipients: [] },
					};
					break;
				case MESSAGING_SERVICE_TYPE.TELEGRAM:
					s = {
						type: MESSAGING_SERVICE_TYPE.TELEGRAM,
						other: { chats: [], token: "" },
					};
					break;
				case MESSAGING_SERVICE_TYPE.EMAIL:
					s = {
						type: MESSAGING_SERVICE_TYPE.EMAIL,
						other: { uri: "" },
					};
					break;
				case MESSAGING_SERVICE_TYPE.SHOUT:
					s = {
						type: MESSAGING_SERVICE_TYPE.SHOUT,
						other: { uri: "" },
					};
					break;
				case MESSAGING_SERVICE_TYPE.NTFY:
					s = {
						type: MESSAGING_SERVICE_TYPE.NTFY,
						other: { uri: "" },
					};
					break;
				default:
					s = {
						type: MESSAGING_SERVICE_TYPE.CUSTOM,
						other: { encoding: MESSAGING_SERVICE_CUSTOM_ENCODING.JSON, send: "" },
					};
			}

			values.push(s);
		},
	},
};
</script>
<style scoped>
h5 {
	position: relative;
	display: flex;
	top: -25px;
	margin-bottom: -0.5rem;
	padding: 0 0.5rem;
	justify-content: center;
}
h5.box-heading {
	top: -34px;
	margin-bottom: -24px;
}
h5 .inner {
	padding: 0 0.5rem;
	background-color: var(--evcc-box);
	font-weight: normal;
	color: var(--evcc-gray);
	text-align: center;
}
</style>
