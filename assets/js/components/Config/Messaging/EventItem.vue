<template>
	<div>
		<div class="form-check form-switch mb-4">
			<input
				:id="formId('switch')"
				:checked="!disabled"
				class="form-check-input form-check-input"
				type="checkbox"
				role="switch"
				:data-testid="`event-${type}-switch`"
				tabindex="0"
				@change="updateDisabled(($event.target as HTMLInputElement).checked)"
			/>
			<label :for="formId('switch')" class="form-check-label fw-bold">
				{{ $t(`config.messaging.event.${type}.title`) }}
			</label>
		</div>
		<div class="row mb-3" :class="{ 'opacity-25': disabled }">
			<div class="col-md-2 col-form-label">
				<label :for="formId('title')">{{ $t("config.messaging.eventTitle") }}</label>
			</div>
			<div class="col-md-10">
				<PropertyField
					:id="formId('title')"
					:model-value="title"
					:data-testid="`event-${type}-title`"
					type="String"
					:disabled="disabled"
					required
					@update:model-value="updateTitle"
				/>
			</div>
		</div>
		<div class="row" :class="{ 'opacity-25': disabled }">
			<div class="col-md-2 col-form-label">
				<label :for="formId('message')">{{ $t("config.messaging.eventMessage") }}</label>
			</div>
			<div class="col-md-10">
				<PropertyField
					:id="formId('message')"
					:model-value="message"
					:data-testid="`event-${type}-message`"
					type="String"
					property="eventMessage"
					:disabled="disabled"
					:rows="3"
					required
					@change="updateMessage($event.target.value)"
				/>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import type { PropType } from "vue";
import { MESSAGING_EVENTS } from "@/types/evcc";
import PropertyField from "../PropertyField.vue";

const EVENT_PARAMS: Record<MESSAGING_EVENTS, Record<string, string>> = {
	asleep: { vehicleName: "{{ if .vehicleTitle }}{{ .vehicleTitle }} {{ end }}" },
	connect: { pvPower: "${pvPower:%.1fk}" },
	disconnect: { connectedDuration: "${connectedDuration}" },
	soc: { vehicleSoc: "${vehicleSoc:%.0f}" },
	start: { mode: "${mode}" },
	stop: { chargedEnergy: "${chargedEnergy:%.1fk}", chargeDuration: "${chargeDuration}" },
	planoverrun: { vehicleTitle: "{{ if .vehicleTitle }} {{ .vehicleTitle }} {{end}}" },
	guest: {},
};

export default {
	name: "EventItem",
	components: { PropertyField },
	props: {
		type: { type: String as PropType<MESSAGING_EVENTS>, required: true },
		disabled: { type: Boolean, required: true },
		title: { type: String, required: true },
		message: { type: String, required: true },
	},
	emits: ["update:disabled", "update:title", "update:message"],
	mounted() {
		if (!this.title) {
			this.updateTitle(this.$t(`config.messaging.event.${this.type}.titleDefault`));
		}

		if (!this.message || this.message === "") {
			const p = EVENT_PARAMS[this.type] || {};
			this.updateMessage(this.$t(`config.messaging.event.${this.type}.messageDefault`, p));
		}
	},
	methods: {
		formId(name: string) {
			return `messaging-event-${this.type}-${name}`;
		},
		updateDisabled(newValue: boolean) {
			this.$emit("update:disabled", !newValue);
		},
		updateTitle(newValue: string) {
			this.$emit("update:title", newValue);
		},
		updateMessage(newValue: string) {
			this.$emit("update:message", newValue);
		},
	},
};
</script>
