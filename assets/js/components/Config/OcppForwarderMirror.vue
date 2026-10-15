<template>
	<div class="border rounded p-3 mb-3" data-testid="ocppforwarder-mirror">
		<div class="d-flex justify-content-between align-items-start gap-2 mb-2">
			<span :class="statusClass" data-testid="ocppforwarder-mirror-status">{{
				statusLabel
			}}</span>
			<button
				type="button"
				class="btn btn-link btn-sm text-danger p-0"
				@click="$emit('remove')"
			>
				{{ $t("config.ocppforwarder.mirrors.remove") }}
			</button>
		</div>
		<div v-if="status?.error" class="alert alert-danger" role="alert">
			{{ status.error }}
		</div>
		<div v-if="status" class="mb-3" data-testid="ocppforwarder-mirror-stats">
			<p class="small mb-1">
				{{ $t("config.ocppforwarder.mirrors.stats.lastSent") }}: {{ lastSentLabel }}
			</p>
			<p class="small text-muted mb-2">{{ summaryLabel }}</p>
			<div
				class="mirror-bars d-flex align-items-end gap-1"
				role="img"
				:aria-label="summaryLabel"
			>
				<div
					v-for="bar in bars"
					:key="bar.start"
					class="mirror-bar flex-fill"
					:class="{ 'mirror-bar--empty': !bar.energyWh }"
					:style="{ height: bar.height + '%' }"
					:title="bar.title"
				/>
			</div>
			<details class="mt-2">
				<summary class="small">{{ $t("config.ocppforwarder.mirrors.stats.table") }}</summary>
				<table class="table table-sm small mb-0">
					<thead>
						<tr>
							<th>{{ $t("config.ocppforwarder.mirrors.stats.hour") }}</th>
							<th class="text-end">
								{{ $t("config.ocppforwarder.mirrors.stats.energy") }}
							</th>
							<th class="text-end">
								{{ $t("config.ocppforwarder.mirrors.stats.messages") }}
							</th>
							<th class="text-end">
								{{ $t("config.ocppforwarder.mirrors.stats.meterValues") }}
							</th>
							<th class="text-end">
								{{ $t("config.ocppforwarder.mirrors.stats.disconnects") }}
							</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="row in rows" :key="row.start">
							<td>{{ row.label }}</td>
							<td class="text-end">{{ row.energy }}</td>
							<td class="text-end">{{ row.messages }}</td>
							<td class="text-end">{{ row.meterValues }}</td>
							<td class="text-end">{{ row.disconnects }}</td>
						</tr>
					</tbody>
				</table>
			</details>
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
				<FormRow
					:id="fieldId('Insecure')"
					:label="$t('config.ocppforwarder.labelInsecure')"
				>
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
import type { OcppMirrorHour, OcppMirrorRule, OcppMirrorSession } from "@/types/evcc";

const HOUR = 3600 * 1000;
const HOURS_SHOWN = 24;

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
		lastSentLabel(): string {
			const { lastSent } = this.status || {};
			return lastSent
				? this.formatTime(lastSent, { dateStyle: "short", timeStyle: "medium" })
				: this.$t("config.ocppforwarder.mirrors.stats.never");
		},
		// the last hours up to now, hours without history count as zero
		hours(): OcppMirrorHour[] {
			const known = new Map(
				(this.status?.history || []).map((h) => [new Date(h.start).getTime(), h])
			);
			const current = Math.floor(Date.now() / HOUR) * HOUR;
			return Array.from({ length: HOURS_SHOWN }, (_, i) => {
				const start = current - (HOURS_SHOWN - 1 - i) * HOUR;
				return (
					known.get(start) || {
						start: new Date(start).toISOString(),
						messages: 0,
						meterValues: 0,
						energyWh: 0,
						disconnects: 0,
					}
				);
			});
		},
		summaryLabel(): string {
			const messages = this.hours.reduce((sum, h) => sum + h.messages, 0);
			const energyWh = this.hours.reduce((sum, h) => sum + h.energyWh, 0);
			return this.$t("config.ocppforwarder.mirrors.stats.last24h", {
				messages,
				energy: this.formatKWh(energyWh),
			});
		},
		bars(): { start: string; height: number; energyWh: number; title: string }[] {
			const max = Math.max(...this.hours.map((h) => h.energyWh), 1);
			return this.hours.map((h) => ({
				start: h.start,
				energyWh: h.energyWh,
				height: Math.max((h.energyWh / max) * 100, 2),
				title: `${this.formatHour(h.start)}: ${this.formatKWh(h.energyWh)} kWh, ${h.messages}`,
			}));
		},
		rows(): {
			start: string;
			label: string;
			energy: string;
			messages: number;
			meterValues: number;
			disconnects: number;
		}[] {
			return this.hours
				.map((h) => ({
					start: h.start,
					label: this.formatHour(h.start),
					energy: this.formatKWh(h.energyWh),
					messages: h.messages,
					meterValues: h.meterValues,
					disconnects: h.disconnects,
				}))
				.reverse();
		},
	},
	methods: {
		fieldId(name: string): string {
			return `ocppforwarderMirror${this.index}${name}`;
		},
		update(field: keyof OcppMirrorRule, value: string | boolean | undefined) {
			this.$emit("update:modelValue", { ...this.modelValue, [field]: value });
		},
		formatTime(value: string, options: Intl.DateTimeFormatOptions): string {
			return new Date(value).toLocaleString(this.$i18n.locale, options);
		},
		formatHour(value: string): string {
			return this.formatTime(value, {
				day: "2-digit",
				month: "short",
				hour: "2-digit",
				minute: "2-digit",
			});
		},
		formatKWh(wh: number): string {
			return (wh / 1e3).toFixed(3);
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

<style scoped>
.mirror-bars {
	height: 4rem;
}
.mirror-bar {
	min-height: 2px;
	border-radius: 2px 2px 0 0;
	background: var(--evcc-green);
}
.mirror-bar--empty {
	background: var(--bs-border-color);
}
</style>
