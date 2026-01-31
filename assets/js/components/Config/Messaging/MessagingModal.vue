<template>
	<div>
		<!-- old ui with yaml editor -->
		<YamlModal
			v-if="messagingConfigured"
			id="messagingModal"
			:title="$t('config.messaging.title')"
			:description="$t('config.messaging.description')"
			docs="/docs/reference/configuration/messaging"
			endpoint="/config/messaging"
			removeKey="messaging"
			data-testid="messaging-modal"
			@changed="$emit('yaml-changed')"
		/>
		<!-- new ui using devices api -->
		<JsonModal
			v-else
			id="messagingModal"
			:title="$t('config.messaging.title')"
			:description="$t('config.messaging.description')"
			docs="/docs/reference/configuration/messaging"
			endpoint="/config/messagingEvents"
			state-key="messagingEvents"
			disable-remove
			data-testid="messaging-modal"
			:size="activeEventsTab ? 'xl' : 'm'"
			:transform-read-values="transformReadValues"
			@changed="$emit('events-changed')"
		>
			<template #default="{ values }: { values: MessagingEvents }">
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
							{{ $t("config.messaging.messengers") }} ({{ messengers.length }})
						</a>
					</li>
				</ul>
				<div v-if="activeEventsTab" class="my-5">
					<div v-for="(event, index) in events" :key="event" class="mb-5">
						<div v-if="values[event]">
							<hr v-if="index > 0" class="my-5" />
							<EventItem
								v-model:disabled="values[event].disabled"
								v-model:title="values[event].title"
								v-model:message="values[event].msg"
								:type="event"
							/>
						</div>
					</div>
				</div>
				<div v-else>
					<div v-for="(m, index) in messengers" :key="index" class="my-4">
						<div
							class="d-flex align-items-center justify-content-between py-2 ps-3 pe-2 border rounded"
						>
							<div class="flex-grow-1">
								<small class="text-muted">{{ m.name }}</small>
								<span class="fw-semibold ms-3">{{ m.type }}</span>
							</div>
							<div class="d-flex">
								<button
									type="button"
									class="btn btn-sm btn-outline-secondary border-0"
									:aria-label="$t('config.messaging.messenger.edit')"
									@click="openMessenger(m.id)"
								>
									<shopicon-regular-edit
										size="s"
										class="flex-shrink-0"
									></shopicon-regular-edit>
								</button>
							</div>
						</div>
					</div>
					<button
						type="button"
						class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
						:aria-label="$t('config.general.remove')"
						tabindex="0"
						@click="openMessenger()"
					>
						<shopicon-regular-plus
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-plus>
						{{ $t("config.messaging.addMessenger") }}
					</button>
				</div>
			</template>
		</JsonModal>
	</div>
</template>

<script lang="ts">
import { MESSAGING_EVENTS, type ConfigMessenger, type MessagingEvents } from "@/types/evcc";
import "@h2d2/shopicons/es/regular/plus";
import "@h2d2/shopicons/es/regular/edit";
import "@h2d2/shopicons/es/regular/arrowright";
import JsonModal from "../JsonModal.vue";
import EventItem from "./EventItem.vue";
import YamlModal from "../YamlModal.vue";
import type { PropType } from "vue";

export default {
	name: "MessagingModal",
	components: {
		YamlModal,
		JsonModal,
		EventItem,
	},
	props: {
		messagingConfigured: Boolean,
		messengers: { type: Array as PropType<ConfigMessenger[]>, required: true },
		selectedMessengerId: Number,
	},
	emits: ["yaml-changed", "events-changed", "open-messenger-modal"],
	data() {
		return {
			activeEventsTab: true,
		};
	},
	computed: {
		events() {
			return Object.values(MESSAGING_EVENTS);
		},
	},
	methods: {
		eventCount(events: MessagingEvents) {
			return Object.values(events).filter((e) => !e.disabled).length;
		},
		transformReadValues(values: MessagingEvents) {
			const v = values ?? {};

			this.events.forEach((event) => {
				const e = v[event];
				v[event] = {
					disabled: e?.disabled ?? true,
					title: e?.title ?? "",
					msg: e?.msg ?? "",
				};
			});

			return v;
		},
		openMessenger(id?: number) {
			this.$emit("open-messenger-modal", id);
		},
	},
};
</script>
