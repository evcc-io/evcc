<template>
	<div class="border rounded p-3 mb-3" data-testid="ocppforwarder-mirror">
		<div class="d-flex justify-content-between align-items-start gap-2 mb-2">
			<span :class="statusClass" data-testid="ocppforwarder-mirror-status">{{
				statusLabel
			}}</span>
			<button type="button" class="btn btn-link btn-sm text-danger p-0" @click="$emit('remove')">
				{{ $t("config.ocppforwarder.mirrors.remove") }}
			</button>
		</div>
		<div v-if="status?.error" class="alert alert-danger" role="alert">
			{{ status.error }}
		</div>
		<FormRow
			:id="fieldId('UpstreamUrl')"
			:label="$t('config.ocppforwarder.upstreamUrl')"
			:help="$t('config.ocppforwarder.upstreamUrlHelp')"
			example="wss://reporting.example.com/ocpp"
		>
			<input
				:id="fieldId('UpstreamUrl')"
				:value="modelValue.upstreamUrl"
				type="text"
				class="form-control"
				inputmode="url"
				spellcheck="false"
				autocomplete="off"
				required
				@input="updateText('upstreamUrl', $event)"
			/>
		</FormRow>
		<FormRow
			:id="fieldId('Username')"
			:label="$t('config.ocppforwarder.username')"
			:help="$t('config.ocppforwarder.usernameHelp')"
			optional
		>
			<input
				:id="fieldId('Username')"
				:value="modelValue.username"
				type="text"
				class="form-control"
				spellcheck="false"
				autocomplete="off"
				@input="updateText('username', $event)"
			/>
		</FormRow>
		<FormRow
			:id="fieldId('Password')"
			:label="$t('config.ocppforwarder.password')"
			:help="$t('config.ocppforwarder.passwordHelp')"
			optional
		>
			<input
				:id="fieldId('Password')"
				:value="modelValue.password"
				type="password"
				class="form-control"
				autocomplete="new-password"
				@input="updateText('password', $event)"
			/>
		</FormRow>
		<FormRow
			:id="fieldId('UpstreamStationId')"
			:label="$t('config.ocppforwarder.upstreamStationId')"
			:help="$t('config.ocppforwarder.upstreamStationIdHelp')"
			optional
		>
			<input
				:id="fieldId('UpstreamStationId')"
				:value="modelValue.upstreamStationId"
				type="text"
				class="form-control"
				spellcheck="false"
				autocomplete="off"
				@input="updateText('upstreamStationId', $event)"
			/>
		</FormRow>
		<PropertyCollapsible>
			<template #advanced>
				<FormRow :id="fieldId('Insecure')" :label="$t('config.ocppforwarder.labelInsecure')">
					<div class="d-flex">
						<input
							:id="fieldId('Insecure')"
							:checked="modelValue.insecure"
							class="form-check-input"
							type="checkbox"
							@change="updateCheckbox('insecure', $event)"
						/>
						<label class="form-check-label ms-2" :for="fieldId('Insecure')">
							{{ $t("config.ocppforwarder.labelCheckInsecure") }}
						</label>
					</div>
				</FormRow>
				<FormRow
					:id="fieldId('CaCert')"
					:label="$t('config.ocppforwarder.labelCaCert')"
					optional
				>
					<PropertyCertField
						:id="fieldId('CaCert')"
						:model-value="modelValue.caCert"
						@update:model-value="update('caCert', $event)"
					/>
				</FormRow>
			</template>
		</PropertyCollapsible>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import FormRow from "./FormRow.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import PropertyCertField from "./PropertyCertField.vue";
import type { OcppMirrorRule, OcppMirrorSession } from "@/types/evcc";

export default defineComponent({
	name: "OcppForwarderMirror",
	components: { FormRow, PropertyCollapsible, PropertyCertField },
	props: {
		modelValue: { type: Object as PropType<OcppMirrorRule>, required: true },
		status: { type: Object as PropType<OcppMirrorSession>, default: undefined },
		index: { type: Number, required: true },
	},
	emits: ["update:modelValue", "remove"],
	computed: {
		connected(): boolean {
			return !!this.status?.upstreamConnected;
		},
		statusClass(): string {
			return this.connected ? "text-success" : "text-muted";
		},
		statusLabel(): string {
			return this.$t(
				this.connected ? "config.ocpp.status.connected" : "config.ocpp.status.configured"
			);
		},
	},
	methods: {
		fieldId(name: string): string {
			return `ocppforwarderMirror${this.index}${name}`;
		},
		update(field: keyof OcppMirrorRule, value: string | boolean | undefined) {
			this.$emit("update:modelValue", { ...this.modelValue, [field]: value });
		},
		updateText(field: keyof OcppMirrorRule, event: Event) {
			this.update(field, (event.target as HTMLInputElement).value);
		},
		updateCheckbox(field: keyof OcppMirrorRule, event: Event) {
			this.update(field, (event.target as HTMLInputElement).checked);
		},
	},
});
</script>
