<template>
	<JsonModal
		id="messagingModal"
		name="messaging"
		:title="$t('config.messaging.title')"
		:description="$t('config.messaging.description')"
		docs="/docs/reference/configuration/messaging"
		endpoint="/config/messagingEvents"
		state-key="messagingEvents"
		data-testid="messaging-modal"
		size="xl"
		:transform-read-values="transformReadValues"
		:disable-save="!activeEventsTab"
		:disable-cancel="!activeEventsTab"
		disable-remove
		@changed="$emit('changed')"
	>
		<template
			#default="{
				values,
				changes,
				save,
			}: {
				values: MessagingEvents;
				changes: boolean;
				save: (shouldClose?: boolean) => Promise<void>;
			}"
		>
			<ul class="nav nav-tabs mb-4">
				<li class="nav-item">
					<a
						class="nav-link tabular"
						:class="{ active: activeEventsTab }"
						href="#"
						@click.prevent="switchToTab(true, changes, save)"
					>
						{{ $t("config.messaging.events") }} ({{ eventCount(values) }})
					</a>
				</li>
				<li class="nav-item">
					<a
						class="nav-link tabular"
						:class="{ active: !activeEventsTab }"
						href="#"
						@click.prevent="switchToTab(false, changes, save)"
					>
						{{ $t("config.messaging.messengers") }} ({{ messengers.length }})
					</a>
				</li>
			</ul>
			<div v-if="activeEventsTab" class="my-5">
				<div class="mb-5">
					<EventItem
						v-for="event in visibleEvents(values)"
						:key="event"
						v-model:disabled="values[event].disabled"
						v-model:title="values[event].title"
						v-model:message="values[event].msg"
						:type="event"
					/>
				</div>
			</div>
			<div v-else>
				<div v-for="(m, index) in messengers" :key="index" class="my-4">
					<div
						class="d-flex align-items-center justify-content-between py-2 px-4 border rounded"
						:data-testid="`messenger-box-${index}`"
					>
						<div class="flex-grow-1">
							<small class="text-muted">#{{ index + 1 }}</small>
							<span class="fw-semibold mx-3">{{ messengerType(m) }}</span>
						</div>
						<DeviceCardEditIcon
							:editable="true"
							:no-edit-button="false"
							@edit="openMessenger(m.id)"
						/>
					</div>
				</div>
				<button
					type="button"
					class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
					tabindex="0"
					@click="openMessenger()"
				>
					<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
					{{ $t("config.messaging.addMessenger") }}
				</button>
			</div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import { MESSAGING_EVENTS, type ConfigMessenger, type MessagingEvents } from "@/types/evcc";
import "@h2d2/shopicons/es/regular/plus";
import JsonModal from "../JsonModal.vue";
import EventItem from "./EventItem.vue";
import { type PropType } from "vue";
import DeviceCardEditIcon from "../DeviceCardEditIcon.vue";
import { capitalize } from "./utils";
import { openModal } from "@/configModal";

export default {
	name: "MessagingModal",
	components: {
		JsonModal,
		EventItem,
		DeviceCardEditIcon,
	},
	props: {
		messengers: { type: Array as PropType<ConfigMessenger[]>, required: true },
	},
	emits: ["changed"],
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
		visibleEvents(values: MessagingEvents) {
			return this.events.filter((e) => values[e]);
		},
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
		async openMessenger(id?: number) {
			await openModal("messenger", { id });
		},
		messengerType(m: ConfigMessenger) {
			const type =
				m.type === "custom" ? this.$t("config.messenger.custom") : m.config.template;
			return capitalize(type ?? "");
		},
		async switchToTab(
			isEventsTab: boolean,
			changes: boolean,
			save: (shouldClose?: boolean) => Promise<void>
		) {
			if (this.activeEventsTab === isEventsTab) return;

			if (this.activeEventsTab && !isEventsTab && changes) {
				if (!window.confirm(this.$t("config.general.confirmSave"))) {
					return;
				}
				await save(false);
			}

			this.activeEventsTab = isEventsTab;
		},
	},
};
</script>
