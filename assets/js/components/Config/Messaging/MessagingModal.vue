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
		:transformReadValues="transformReadValues"
		@changed="$emit('changed')"
	>
		<template #default="{ values }: { values: Messaging }">
			<ul class="nav nav-tabs mb-4">
				<li class="nav-item">
					<a
						class="nav-link tabular"
						:class="{ active: activeEventsTab }"
						href="#"
						@click.prevent="activeEventsTab = true"
					>
						{{ $t("config.messaging.events") }} ({{ eventCount(values) }})
					</a>
				</li>
				<li class="nav-item">
					<a
						class="nav-link tabular"
						:class="{ active: !activeEventsTab }"
						href="#"
						@click.prevent="activeEventsTab = false"
					>
						{{ $t("config.messaging.services") }} ({{ servicesCount(values) }})
					</a>
				</li>
			</ul>
			<div v-if="activeEventsTab" class="my-5">
				<div v-if="values.events">
					<div
						v-for="(event, index) in Object.values(MESSAGING_EVENTS)"
						:key="event"
						class="mb-5"
					>
						<hr v-if="index > 0" class="my-5" />
						<EventItem
							v-model:disabled="values.events[event].disabled"
							v-model:title="values.events[event].title"
							v-model:message="values.events[event].msg"
							:type="event"
						/>
					</div>
				</div>
			</div>
			<div v-else>
				<div v-for="(s, index) in values.services" :key="index" class="my-5">
					<div class="border rounded px-3 pt-4 pb-3 mb-3">
						<div class="d-lg-block">
							<h5 class="box-heading">
								<div class="inner">
									{{ $t("config.messaging.serviceName") }} #{{ index + 1 }} -
									{{ $t(`config.messaging.service.${s.type}.title`) }}
								</div>
							</h5>
						</div>
						<div :data-testid="`service-box-${s.type.toLowerCase()}`">
							<CustomService
								v-if="s.type === MESSAGING_SERVICE_TYPE.CUSTOM"
								v-model:encoding="s.other.encoding"
								v-model:send="s.other.send"
							/>
							<EmailService
								v-else-if="s.type === MESSAGING_SERVICE_TYPE.EMAIL"
								v-model:host="s.other.host"
								v-model:port="s.other.port"
								v-model:user="s.other.user"
								v-model:password="s.other.password"
								v-model:from="s.other.from"
								v-model:to="s.other.to"
							/>
							<NtfyService
								v-else-if="s.type === MESSAGING_SERVICE_TYPE.NTFY"
								v-model:host="s.other.host"
								v-model:topics="s.other.topics"
								v-model:priority="s.other.priority"
								v-model:tags="s.other.tags"
								v-model:authtoken="s.other.authtoken"
							/>
							<PushoverService
								v-else-if="s.type === MESSAGING_SERVICE_TYPE.PUSHOVER"
								v-model:app="s.other.app"
								v-model:recipients="s.other.recipients"
								v-model:devices="s.other.devices"
							/>
							<ShoutService
								v-else-if="s.type === MESSAGING_SERVICE_TYPE.SHOUT"
								v-model:uri="s.other.uri"
							/>
							<TelegramService
								v-else
								v-model:token="s.other.token"
								v-model:chats="s.other.chats"
							/>
						</div>
					</div>

					<button
						type="button"
						class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray ms-auto"
						:aria-label="$t('config.general.remove')"
						tabindex="0"
						@click="removeService(values, s)"
					>
						<shopicon-regular-trash
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-trash>
						{{ $t("config.general.remove") }}
					</button>
				</div>
				<hr v-if="servicesCount(values) > 0" class="mb-5" />
				<DropdownButton
					:actions="dropDownActions"
					@click="(t: MESSAGING_SERVICE_TYPE) => addService(values, t)"
				>
					<div class="d-flex align-items-center gap-2">
						<shopicon-regular-plus
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-plus>
						{{ $t("config.messaging.addService") }}
					</div>
				</DropdownButton>
			</div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import {
	MESSAGING_EVENTS,
	MESSAGING_SERVICE_TYPE,
	MESSAGING_SERVICE_NTFY_PRIORITY,
	type Messaging,
	type MessagingServices,
	MESSAGING_SERVICE_CUSTOM_ENCODING,
	type SelectActionOption,
} from "@/types/evcc";
import defaultYaml from ".././defaultYaml/messaging.yaml?raw";
import "@h2d2/shopicons/es/regular/plus";
import "@h2d2/shopicons/es/regular/trash";
import "@h2d2/shopicons/es/regular/arrowright";
import deepEqual from "@/utils/deepEqual";
import JsonModal from "../JsonModal.vue";
import PushoverService from "./Services/PushoverService.vue";
import TelegramService from "./Services/TelegramService.vue";
import EmailService from "./Services/EmailService.vue";
import ShoutService from "./Services/ShoutService.vue";
import NtfyService from "./Services/NtfyService.vue";
import EventItem from "./EventItem.vue";
import DropdownButton from "@/components/Helper/DropdownButton.vue";
import CustomService from "./Services/CustomService.vue";

export default {
	name: "MessagingModal",
	components: {
		JsonModal,
		PushoverService,
		TelegramService,
		EmailService,
		ShoutService,
		NtfyService,
		CustomService,
		EventItem,
		DropdownButton,
	},
	emits: ["changed"],
	data() {
		return {
			defaultYaml: defaultYaml.trim(),
			MESSAGING_EVENTS,
			MESSAGING_SERVICE_TYPE,
			MESSAGING_SERVICE_NTFY_PRIORITY,
			activeEventsTab: true,
			deepEqual,
		};
	},
	computed: {
		dropDownActions(): SelectActionOption<string>[] {
			return Object.values(MESSAGING_SERVICE_TYPE).map((s) => ({
				value: s,
				name: this.$t(`config.messaging.service.${s}.title`),
			}));
		},
	},
	methods: {
		eventCount(values: Messaging) {
			return Object.values(values.events ?? {}).filter((e) => !e.disabled).length;
		},
		servicesCount(values: Messaging) {
			return values.services?.length ?? 0;
		},
		removeService(values: Messaging, service: MessagingServices) {
			const index = values.services?.findIndex((f) => deepEqual(service, f));
			if (index !== undefined && index !== -1) {
				values.services?.splice(index, 1);
			}
		},
		addService(values: Messaging, type: MESSAGING_SERVICE_TYPE) {
			values.services = [...(values.services ?? []), this.createService(type)];
		},
		getServiceComponent(type: MESSAGING_SERVICE_TYPE) {
			switch (type) {
				case MESSAGING_SERVICE_TYPE.PUSHOVER:
					return "PushoverService";
				case MESSAGING_SERVICE_TYPE.TELEGRAM:
					return "TelegramService";
				case MESSAGING_SERVICE_TYPE.EMAIL:
					return "EmailService";
				case MESSAGING_SERVICE_TYPE.SHOUT:
					return "ShoutService";
				case MESSAGING_SERVICE_TYPE.NTFY:
					return "NtfyService";
				default:
					return "CustomService";
			}
		},
		transformReadValues(values: Messaging) {
			const v = values ?? {};

			if (!v.events) {
				v.events = {} as Messaging["events"];
			}

			Object.values(MESSAGING_EVENTS).forEach((evt) => {
				const e = v.events![evt];
				v.events![evt] = {
					disabled: e?.disabled ?? true,
					title: e?.title ?? "",
					msg: e?.msg ?? "",
				};
			});

			return v;
		},
		createService(serviceType: MESSAGING_SERVICE_TYPE): MessagingServices {
			switch (serviceType) {
				case MESSAGING_SERVICE_TYPE.PUSHOVER:
					return {
						type: MESSAGING_SERVICE_TYPE.PUSHOVER,
						other: { app: "", devices: [], recipients: [] },
					};
				case MESSAGING_SERVICE_TYPE.TELEGRAM:
					return {
						type: MESSAGING_SERVICE_TYPE.TELEGRAM,
						other: { chats: [], token: "" },
					};

				case MESSAGING_SERVICE_TYPE.EMAIL:
					return {
						type: MESSAGING_SERVICE_TYPE.EMAIL,
						other: { host: "", port: 465, user: "", password: "", from: "", to: "" },
					};

				case MESSAGING_SERVICE_TYPE.SHOUT:
					return {
						type: MESSAGING_SERVICE_TYPE.SHOUT,
						other: { uri: "" },
					};

				case MESSAGING_SERVICE_TYPE.NTFY:
					return {
						type: MESSAGING_SERVICE_TYPE.NTFY,
						other: { host: "ntfy.sh", topics: [] },
					};

				default:
					return {
						type: MESSAGING_SERVICE_TYPE.CUSTOM,
						other: { encoding: MESSAGING_SERVICE_CUSTOM_ENCODING.JSON, send: "" },
					};
			}
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
