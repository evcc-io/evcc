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
				<div v-for="(s, index) in values.services" :key="index">
					<div class="d-block">
						<hr class="mt-5" />
						<h5>
							<div class="inner mb-4">{{ s.type }}</div>
						</h5>
					</div>

					<div class="border rounded px-3 pt-4 pb-3">
						<div v-if="s.type === MESSAGING_SERVICE_TYPE.PUSHOVER">
							<FormRow :id="'messagingServicePushoverApp'" label="App">
								<PropertyField
									:id="'messagingServicePushoverApp'"
									v-model="s.other.app"
									type="String"
									required
								/>
							</FormRow>
							<FormRow :id="'messagingServicePushoverApp'" label="App">
								<PropertyField
									:id="'messagingServicePushoverApp'"
									v-model="s.other.app"
									type="String"
									required
								/>
							</FormRow>
						</div>
						<div v-if="s.type === MESSAGING_SERVICE_TYPE.TELEGRAM"></div>
						<div v-if="s.type === MESSAGING_SERVICE_TYPE.EMAIL"></div>
						<div v-if="s.type === MESSAGING_SERVICE_TYPE.SHOUT"></div>
						<div v-if="s.type === MESSAGING_SERVICE_TYPE.NTFY"></div>
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
import { MESSAGING_EVENTS, MESSAGING_SERVICE_TYPE, type Messaging } from "@/types/evcc";
import JsonModal from "./JsonModal.vue";
import defaultYaml from "./defaultYaml/messaging.yaml?raw";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";

export default {
	name: "MessagingModal",
	components: { JsonModal, FormRow, PropertyField },
	emits: ["changed"],
	data() {
		return {
			defaultYaml: defaultYaml.trim(),
			MESSAGING_EVENTS,
			MESSAGING_SERVICE_TYPE,
			eventsTabActive: false,
		};
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
