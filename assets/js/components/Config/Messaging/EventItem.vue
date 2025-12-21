<template>
	<div>
		<div class="d-flex align-items-center mb-3">
			<div class="form-switch me-2">
				<input
					:checked="!disabled"
					class="form-check-input"
					type="checkbox"
					role="switch"
					:data-testid="`event-${type}-switch`"
					tabindex="0"
					@change="updateDisabled(($event.target as HTMLInputElement).checked)"
				/>
			</div>
			<h6 class="my-0">{{ $t(`config.messaging.event.${type}.title`) }}</h6>
		</div>
		<div class="container"></div>
		<div class="row mb-3">
			<div class="col-2 col-form-label">
				<label :for="formId('title')"> Title </label>
			</div>
			<div class="col-10">
				<PropertyField
					:id="formId('title')"
					:model-value="title"
					:data-testid="`event-${type}-title`"
					type="String"
					:disabled="disabled"
					required
					@change="updateTitle($event.target.value)"
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
					:model-value="message"
					:data-testid="`event-${type}-message`"
					type="String"
					property="eventMessage"
					:disabled="disabled"
					required
					rows
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
			let p = {};

			switch (this.type) {
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
				case MESSAGING_EVENTS.PLANOVERRUN:
					p = {
						vehicleTitle: "{{ if .vehicleTitle }} {{ .vehicleTitle }} {{end}}",
					};
					break;
			}

			this.updateMessage(this.$t(`config.messaging.event.${this.type}.messageDefault`, p));
		}
	},
	methods: {
		formId(name: string) {
			return `messaging-event-${this.type}-${name}`;
		},
		updateDisabled(newValue: boolean) {
			this.$emit("update:disabled", newValue);
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
