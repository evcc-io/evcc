<template>
	<div>
		<div class="d-flex align-items-center mb-3">
			<div class="form-switch me-2">
				<input
					v-model="eventObjectData.enabled"
					class="form-check-input"
					type="checkbox"
					role="switch"
					:data-testid="`event-${eventType}-switch`"
					tabindex="0"
				/>
			</div>
			<h6 class="my-0">{{ $t(`config.messaging.event.${eventType}.title`) }}</h6>
		</div>
		<div class="container"></div>
		<div class="row mb-3">
			<div class="col-2 col-form-label">
				<label :for="formId('title')"> Title </label>
			</div>
			<div class="col-10">
				<PropertyField
					:id="formId('title')"
					v-model="eventObjectData.title"
					:data-testid="`event-${eventType}-title`"
					type="String"
					:disabled="!eventObjectData.enabled"
					required
				/>
			</div>
		</div>
		<div class="row">
			<div class="col-2 col-form-label">
				<label :for="formId('message')"> Message </label>
			</div>
			<div class="col-10">
				<PropertyField
					:id="formId('message')"
					v-model="eventObjectData.msg"
					:data-testid="`event-${eventType}-message`"
					type="String"
					property="eventMessage"
					:disabled="!eventObjectData.enabled"
					required
				/>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import type { PropType } from "vue";
import PropertyField from "../../PropertyField.vue";
import { MESSAGING_EVENTS, type MessagingEvent } from "@/types/evcc";

export default {
	name: "EventItem",
	components: { PropertyField },
	props: {
		eventType: { type: String as PropType<MESSAGING_EVENTS>, required: true },
		eventObject: {
			type: Object as () => MessagingEvent,
			required: true,
		},
	},
	data() {
		return {
			eventObjectData: this.eventObject,
		};
	},
	mounted() {
		if (!this.eventObjectData.title) {
			this.eventObjectData.title = this.$t(
				`config.messaging.event.${this.eventType}.titleDefault`
			);
		}

		if (!this.eventObjectData.msg) {
			let p = {};

			switch (this.eventType) {
				case MESSAGING_EVENTS.ASLEEP:
					p = { vehicleName: "{{ if .vehicleTitle }}{{ .vehicleTitle }} {{ end }}" };
					break;
				case MESSAGING_EVENTS.CONNECT:
					p = { pvPower: "${pvPower:%.1fk}" };
					break;
				case MESSAGING_EVENTS.DISCONNECT:
					p = { connectedDuration: "${connectedDuration}" };
					break;
				case MESSAGING_EVENTS.SOC:
					p = { vehicleSoc: "${vehicleSoc:%.0f}" };
					break;
				case MESSAGING_EVENTS.START:
					p = { mode: "${mode}" };
					break;
				case MESSAGING_EVENTS.STOP:
					p = {
						chargedEnergy: "${chargedEnergy:%.1fk}",
						chargeDuration: "${chargeDuration}",
					};
					break;
			}

			this.eventObjectData.msg = this.$t(
				`config.messaging.event.${this.eventType}.messageDefault`,
				p
			);
		}
	},
	methods: {
		formId(name: string) {
			return `messaging-event-${this.eventType}-${name}`;
		},
	},
};
</script>
