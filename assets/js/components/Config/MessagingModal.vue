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
						:class="{ active: activeEventsTab }"
						href="#"
						@click.prevent="activeEventsTab = true"
					>
						Events
					</a>
				</li>
				<li class="nav-item">
					<a
						class="nav-link"
						:class="{ active: !activeEventsTab }"
						href="#"
						@click.prevent="activeEventsTab = false"
					>
						Services
					</a>
				</li>
			</ul>
			<div v-if="activeEventsTab">
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
				<div v-for="(s, index) in values.services" :key="index" class="mb-5">
					<div class="border rounded px-3 pt-4 pb-3 mb-3">
						<div class="d-lg-block">
							<h5 class="box-heading">
								<div class="inner">Messaging #{{ index + 1 }} - {{ s.type }}</div>
							</h5>
						</div>

						<component :is="getServiceComponent(s.type)" :service="s" />
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
				<hr v-if="values.services && values.services?.length != 0" class="mb-5" />
				<div class="d-flex align-items-center mb-5">
					<button
						type="button"
						class="d-flex btn btn-sm align-items-center gap-2 btn-outline-secondary border-0 evcc-gray"
						data-testid="networkconnection-add"
						tabindex="0"
						@click="values.services = [...(values.services ?? []), addMessaging()]"
					>
						<shopicon-regular-plus
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-plus>
						Add messaging:
					</button>
					<CustomSelect
						id="messagingSelectedServiceAdd"
						class="options"
						:options="serviceOptions"
						:selected="selectedService"
						@change="selectedService = $event.target.value"
					>
						<span class="evcc-gray flex-grow-1 text-truncate select-service">
							{{ selectedService }}
						</span>
					</CustomSelect>
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
	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	type Messaging,
	type MessagingServices,
	MESSAGING_SERVICE_CUSTOM_ENCODING,
	type SelectOption,
} from "@/types/evcc";
import JsonModal from "./JsonModal.vue";
import defaultYaml from "./defaultYaml/messaging.yaml?raw";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import YamlEntry from "./DeviceModal/YamlEntry.vue";
import PushoverService from "./Services/PushoverService.vue";
import TelegramService from "./Services/TelegramService.vue";
import EmailService from "./Services/EmailService.vue";
import ShoutService from "./Services/ShoutService.vue";
import NftyService from "./Services/NftyService.vue";
import CustomService from "./Services/CustomService.vue";
import "@h2d2/shopicons/es/regular/plus";
import "@h2d2/shopicons/es/regular/trash";
import "@h2d2/shopicons/es/regular/arrowright";
import deepEqual from "@/utils/deepEqual";
import CustomSelect from "../Helper/CustomSelect.vue";

export default {
	name: "MessagingModal",
	components: {
		JsonModal,
		FormRow,
		PropertyField,
		YamlEntry,
		PushoverService,
		TelegramService,
		EmailService,
		ShoutService,
		NftyService,
		CustomService,
		CustomSelect,
	},
	emits: ["changed"],
	data() {
		return {
			defaultYaml: defaultYaml.trim(),
			MESSAGING_EVENTS,
			MESSAGING_SERVICE_TYPE,
			MESSAGING_SERVICE_NFTY_PRIORITY,
			activeEventsTab: true,
			deepEqual,
			selectedService: MESSAGING_SERVICE_TYPE.EMAIL,
		};
	},
	computed: {
		serviceOptions(): SelectOption<string>[] {
			return Object.values(MESSAGING_SERVICE_TYPE).map((s) => ({
				value: s,
				name: s,
			}));
		},
	},
	methods: {
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
					return "NftyService";
				default:
					return "CustomService";
			}
		},
		addMessaging(): MessagingServices {
			switch (this.selectedService) {
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
						other: { uri: "" },
					};

				case MESSAGING_SERVICE_TYPE.SHOUT:
					return {
						type: MESSAGING_SERVICE_TYPE.SHOUT,
						other: { uri: "" },
					};

				case MESSAGING_SERVICE_TYPE.NTFY:
					return {
						type: MESSAGING_SERVICE_TYPE.NTFY,
						other: { uri: "" },
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

.options .select-service {
	text-decoration: underline;
}
</style>
