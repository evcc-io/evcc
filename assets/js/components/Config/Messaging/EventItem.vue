<template>
	<div
		:data-testid="`event-${type}`"
		role="group"
		:aria-label="type"
		class="collapsible-wrapper mb-2"
		:class="{ open: !disabled }"
	>
		<div class="form-check form-switch mb-4">
			<input
				:id="formId('switch')"
				:checked="!disabled"
				class="form-check-input form-check-input"
				type="checkbox"
				role="switch"
				tabindex="0"
				@change="updateDisabled(($event.target as HTMLInputElement).checked)"
			/>
			<label :for="formId('switch')" class="form-check-label fw-bold">
				{{ $t(`config.messaging.event.${type}.title`) }}
			</label>
		</div>
		<div class="collapsible-content">
			<div class="row mb-3">
				<div class="col-md-2 col-form-label">
					<label :for="formId('title')">{{ $t("config.messaging.eventTitle") }}</label>
				</div>
				<div class="col-md-10">
					<PropertyField
						:id="formId('title')"
						:model-value="title"
						type="String"
						:disabled="disabled"
						required
						@update:model-value="updateTitle"
					/>
				</div>
			</div>
			<div class="row mb-5">
				<div class="col-md-2 col-form-label">
					<label :for="formId('message')">{{
						$t("config.messaging.eventMessage")
					}}</label>
				</div>
				<div class="col-md-10">
					<PropertyField
						:id="formId('message')"
						:model-value="message"
						type="String"
						property="eventMessage"
						:disabled="disabled"
						:rows="3"
						required
						@change="updateMessage($event.target.value)"
					/>
					<div class="text-end small text-gray mt-1">
						<a
							:href="`${docsPrefix()}/docs/reference/configuration/messaging#msg`"
							target="_blank"
							class="text-gray"
						>
							{{ $t("config.messaging.seePlaceholders") }}
						</a>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import type { PropType } from "vue";
import { MESSAGING_EVENTS } from "@/types/evcc";
import PropertyField from "../PropertyField.vue";
import { docsPrefix } from "@/i18n";

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
		docsPrefix,
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

<style scoped>
.collapsible-wrapper {
	grid-template-rows: auto 0fr;
}

.collapsible-wrapper.open {
	grid-template-rows: auto 1fr;
}
</style>
