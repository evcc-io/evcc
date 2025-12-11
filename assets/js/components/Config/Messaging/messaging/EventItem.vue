<template>
	<div>
		<div class="d-flex align-items-center mb-3">
			<div class="form-switch me-2">
				<input
					class="form-check-input"
					type="checkbox"
					role="switch"
					data-testid="static-plan-active"
					v-model="eventData.enabled"
					tabindex="0"
				/>
			</div>
			<h6 class="my-0">{{ $t("messaging.event." + eventKey) }}</h6>
		</div>
		<div class="container"></div>
		<div class="row mb-3">
			<div class="col-2 col-form-label">
				<label :for="formId('title')"> Title </label>
			</div>
			<div class="col-10">
				<PropertyField
					:id="formId('title')"
					v-model="eventData.title"
					type="String"
					:disabled="!eventData.enabled"
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
					v-model="eventData.msg"
					type="String"
					property="eventMessage"
					:disabled="!eventData.enabled"
					required
				/>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import type { PropType } from "vue";
import FormRow from "../../FormRow.vue";
import PropertyField from "../../PropertyField.vue";
import { MESSAGING_EVENTS, type Messaging, type MessagingEvent } from "@/types/evcc";

export default {
	name: "EventItem",
	components: { FormRow, PropertyField },
	props: {
		eventKey: { type: String as PropType<MESSAGING_EVENTS>, required: true },
		values: { type: Object as () => Messaging, required: true },
	},
	computed: {
		// TODO: remove
		eventData(): MessagingEvent {
			const evs = this.values.events as any;
			if (!evs) {
				(this.values as any).events = {};
			}
			const key = this.eventKey as unknown as string;
			if (!(this.values.events as any)[key])
				(this.values.events as any)[key] = { title: "", msg: "", enabled: true };
			return (this.values.events as any)[key];
		},
	},
	methods: {
		formId(name: string) {
			return `messaging-event-${this.eventKey}-${name}`;
		},
	},
};
</script>
