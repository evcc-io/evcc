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
						:class="{ active: !eventsTabActive }"
						href="#"
						@click.prevent="eventsTabActive = false"
					>
						Services
					</a>
				</li>
				<li class="nav-item">
					<a
						class="nav-link"
						:class="{ active: eventsTabActive }"
						href="#"
						@click.prevent="eventsTabActive = true"
					>
						Events
					</a>
				</li>
			</ul>

			<div v-show="!eventsTabActive">
				<div v-for="(s, index) in values.services" :key="index" class="mb-5">
					<div class="border rounded px-3 pt-4 pb-3">
						<div class="d-lg-block">
							<h5 class="box-heading">
								<div class="inner">{{ s.type }}</div>
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
						<div v-if="s.type === MESSAGING_SERVICE_TYPE.TELEGRAM">
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
						<div v-if="s.type === MESSAGING_SERVICE_TYPE.EMAIL">
							{{ s.other.uri }}
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
						<div v-if="s.type === MESSAGING_SERVICE_TYPE.SHOUT">
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
						<div v-if="s.type === MESSAGING_SERVICE_TYPE.NTFY">
							<FormRow
								id="messagingServiceNftyHost"
								label="Host"
								:help="$t('config.messaging.nfty.host')"
							>
								<PropertyField
									id="messagingServiceNftyHost"
									:model-value="decodeNfty(s.other.uri)[NFTY_PROPERTY.HOST]"
									@update:model-value="
										(e) => changeNftyValue(s.other.uri, NFTY_PROPERTY.HOST, e)
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
										decodeNfty(s.other.uri)[NFTY_PROPERTY.TOPICS].replace(
											',',
											'\n'
										)
									"
									@update:model-value="
										(e) =>
											changeNftyValue(
												s.other.uri,
												NFTY_PROPERTY.TOPICS,
												e
											).replace('\n', ',')
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
							>
								<PropertyField
									id="messagingServiceNftyPriority"
									property="priority"
									type="Choice"
									class="me-2 w-25"
									:choice="Object.values(MESSAGING_SERVICE_NFTY_PRIORITY)"
									required
									:model-value="
										s.other.priority || MESSAGING_SERVICE_NFTY_PRIORITY.DEFAULT
									"
									@update:model-value="s.other.priority = $event.target.value"
								/>
							</FormRow>
							<FormRow
								id="messagingServiceNftyTags"
								label="Recipients"
								:help="$t('config.messaging.nfty.tags')"
							>
								<PropertyField
									id="messagingServiceNftyTags"
									:model-value="s.other.tags?.replace(',', '\n')"
									@update:model-value="
										(e) => (s.other.tags = e.replace('\n', ','))
									"
									property="tags"
									type="List"
									required
								/>
							</FormRow>
						</div>
						<div v-else></div>
					</div>

					<button
						type="button"
						class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray ms-auto"
						:aria-label="$t('config.general.remove')"
						tabindex="0"
						@click="values.services?.splice(index, 1)"
					>
						<shopicon-regular-trash
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-trash>
						{{ $t("config.general.remove") }}
					</button>
				</div>
			</div>

			<div v-show="eventsTabActive">
				<div v-for="event in MESSAGING_EVENTS" :key="event">
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
		</template>
	</JsonModal>
</template>

<script lang="ts">
import {
	MESSAGING_EVENTS,
	MESSAGING_SERVICE_TYPE,
	MESSAGING_SERVICE_NFTY_PRIORITY,
	type Messaging,
} from "@/types/evcc";
import JsonModal from "./JsonModal.vue";
import defaultYaml from "./defaultYaml/messaging.yaml?raw";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";

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
	components: { JsonModal, FormRow, PropertyField },
	emits: ["changed"],
	data() {
		return {
			defaultYaml: defaultYaml.trim(),
			MESSAGING_EVENTS,
			MESSAGING_SERVICE_TYPE,
			EMAIL_PROPERTY,
			MESSAGING_SERVICE_NFTY_PRIORITY,
			NFTY_PROPERTY,
			eventsTabActive: false,
		};
	},
	methods: {
		decodeEmail(uri: string) {
			const url = new URL(uri.replace(/^smtp/, "http"));
			const params = new URLSearchParams(url.search);

			return {
				[EMAIL_PROPERTY.HOST]: url.hostname,
				[EMAIL_PROPERTY.PORT]: url.port,
				[EMAIL_PROPERTY.USER]: url.username,
				[EMAIL_PROPERTY.PASSWORD]: url.password,
				[EMAIL_PROPERTY.FROM]: params.get("fromAddress"),
				[EMAIL_PROPERTY.TO]: params.get("toAddresses"),
			};
		},
		changeEmailValue(uri: string, p: EMAIL_PROPERTY, v: string) {
			var d = this.decodeEmail(uri);
			d[p] = v;
			return `smtp://${d[EMAIL_PROPERTY.USER]}:${d[EMAIL_PROPERTY.PASSWORD]}@${d[EMAIL_PROPERTY.HOST]}:${d[EMAIL_PROPERTY.PORT]}/?fromAddress=${d[EMAIL_PROPERTY.FROM]}&toAddresses=${d[EMAIL_PROPERTY.TO]}`;
		},
		decodeNfty(uri: string) {
			const url = new URL(uri);
			const host = url.hostname;
			const topics = url.pathname.endsWith("/") ? url.pathname.slice(1) : url.pathname;

			return {
				[NFTY_PROPERTY.HOST]: host,
				[NFTY_PROPERTY.TOPICS]: topics,
			};
		},
		changeNftyValue(uri: string, p: NFTY_PROPERTY, v: string) {
			var d = this.decodeNfty(uri);
			d[p] = v;
			return `https://${d[NFTY_PROPERTY.HOST]}/${d[NFTY_PROPERTY.TOPICS]}`;
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
