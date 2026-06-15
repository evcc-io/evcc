<template>
	<JsonModal
		name="ocppforwarder"
		:title="$t('config.ocppforwarder.editTitle')"
		:description="$t('config.ocppforwarder.description')"
		endpoint="/config/ocppforwarder"
		state-key="ocppforwarder.config"
		no-buttons
		:transform-read-values="transformReadValues"
		:transform-write-values="transformWriteValues"
		@changed="$emit('changed')"
	>
		<template
			#default="{
				values,
				changes,
				save,
			}: {
				values: OcppForwarderRule;
				changes: boolean;
				save: (close?: boolean) => void;
			}"
		>
			<p v-if="ruleExists" class="mb-3" data-testid="ocppforwarder-status">
				{{ $t("config.ocppforwarder.status") }}:
				<span :class="connectionConnected ? 'text-success' : 'text-danger'">{{
					connectionLabel
				}}</span>
			</p>
			<div
				v-if="sessionError"
				class="alert alert-danger"
				role="alert"
				data-testid="ocppforwarder-error"
			>
				{{ sessionError }}
			</div>
			<FormRow
				id="ocppforwarderUpstreamUrl"
				:label="$t('config.ocppforwarder.upstreamUrl')"
				:help="$t('config.ocppforwarder.upstreamUrlHelp')"
				example="wss://billing.example.com/ocpp"
			>
				<input
					id="ocppforwarderUpstreamUrl"
					v-model="values.upstreamUrl"
					type="text"
					class="form-control"
					inputmode="url"
					spellcheck="false"
					autocomplete="off"
					required
				/>
			</FormRow>
			<FormRow
				id="ocppforwarderUsername"
				:label="$t('config.ocppforwarder.username')"
				:help="$t('config.ocppforwarder.usernameHelp')"
				optional
			>
				<input
					id="ocppforwarderUsername"
					v-model="values.username"
					type="text"
					class="form-control"
					spellcheck="false"
					autocomplete="off"
				/>
			</FormRow>
			<FormRow
				id="ocppforwarderPassword"
				:label="$t('config.ocppforwarder.password')"
				:help="$t('config.ocppforwarder.passwordHelp')"
				optional
			>
				<input
					id="ocppforwarderPassword"
					v-model="values.password"
					type="password"
					class="form-control"
					autocomplete="new-password"
				/>
			</FormRow>
			<FormRow
				id="ocppforwarderUpstreamStationId"
				:label="$t('config.ocppforwarder.upstreamStationId')"
				:help="$t('config.ocppforwarder.upstreamStationIdHelp')"
				optional
			>
				<input
					id="ocppforwarderUpstreamStationId"
					v-model="values.upstreamStationId"
					type="text"
					class="form-control"
					:placeholder="targetStationId"
					spellcheck="false"
					autocomplete="off"
				/>
			</FormRow>
			<PropertyCollapsible>
				<template #advanced>
					<FormRow
						id="ocppforwarderReadOnly"
						:label="$t('config.ocppforwarder.readOnly.label')"
						:help="getReadOnlyHelp(values.readOnly)"
					>
						<div class="d-flex">
							<input
								id="ocppforwarderReadOnly"
								v-model="values.readOnly"
								class="form-check-input"
								type="checkbox"
							/>
							<label class="form-check-label ms-2" for="ocppforwarderReadOnly">
								{{ $t("config.ocppforwarder.readOnly.check") }}
							</label>
						</div>
					</FormRow>
					<FormRow
						id="ocppforwarderInsecure"
						:label="$t('config.ocppforwarder.labelInsecure')"
					>
						<div class="d-flex">
							<input
								id="ocppforwarderInsecure"
								v-model="values.insecure"
								class="form-check-input"
								type="checkbox"
							/>
							<label class="form-check-label ms-2" for="ocppforwarderInsecure">
								{{ $t("config.ocppforwarder.labelCheckInsecure") }}
							</label>
						</div>
					</FormRow>
					<FormRow
						id="ocppforwarderCaCert"
						:label="$t('config.ocppforwarder.labelCaCert')"
						optional
					>
						<PropertyCertField id="ocppforwarderCaCert" v-model="values.caCert" />
					</FormRow>
				</template>
			</PropertyCollapsible>

			<div class="mt-4 d-flex justify-content-between gap-2 flex-column flex-sm-row">
				<div
					class="d-flex justify-content-between order-2 order-sm-1 gap-2 flex-grow-1 flex-sm-grow-0"
				>
					<button
						type="button"
						class="btn btn-link text-muted btn-cancel"
						data-bs-dismiss="modal"
					>
						{{ $t("config.general.cancel") }}
					</button>
					<button
						v-if="ruleExists"
						type="button"
						class="btn btn-link text-danger"
						:disabled="removing"
						@click="removeRule"
					>
						{{ $t("config.general.remove") }}
					</button>
				</div>
				<button
					v-if="changes"
					type="button"
					class="btn btn-primary order-1 order-sm-2 flex-grow-1 flex-sm-grow-0 px-4"
					:disabled="!values.upstreamUrl"
					@click="save(false)"
				>
					{{ $t("config.general.save") }}
				</button>
				<button
					v-else
					type="button"
					class="btn btn-outline-primary order-1 order-sm-2 flex-grow-1 flex-sm-grow-0 px-4"
					data-bs-dismiss="modal"
				>
					{{ $t("config.general.close") }}
				</button>
			</div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import PropertyCertField from "./PropertyCertField.vue";
import type { OcppForwarderRule, OcppForwarderSession } from "@/types/evcc";
import { getModal, closeModal } from "@/configModal";
import api from "@/api";
import store from "@/store";

export default defineComponent({
	name: "OcppForwarderModal",
	components: { JsonModal, FormRow, PropertyCollapsible, PropertyCertField },
	emits: ["changed"],
	data() {
		return { removing: false };
	},
	computed: {
		// station id the modal is editing, carried via the config modal stack
		targetStationId(): string {
			return getModal("ocppforwarder")?.station || "";
		},
		rules(): OcppForwarderRule[] {
			return store.state?.ocppforwarder?.config || [];
		},
		ruleExists(): boolean {
			return this.rules.some((r) => r.stationId === this.targetStationId);
		},
		session(): OcppForwarderSession | undefined {
			return store.state?.ocppforwarder?.status?.find(
				(s) => s.chargerId === this.targetStationId
			);
		},
		sessionError(): string | undefined {
			return this.session?.error;
		},
		connectionConnected(): boolean {
			return !!this.session?.upstreamConnected;
		},
		connectionLabel(): string {
			return this.$t(
				this.connectionConnected
					? "config.ocpp.status.connected"
					: "config.ocpp.status.configured"
			);
		},
	},
	methods: {
		getReadOnlyHelp(readOnly?: boolean): string {
			return this.$t(`config.ocppforwarder.readOnly.help.${readOnly ? "true" : "false"}`);
		},
		// pick the rule for the target station, or seed a new one prefilled with the station id
		transformReadValues(rules: OcppForwarderRule[]): OcppForwarderRule {
			const list = Array.isArray(rules) ? rules : [];
			const existing = list.find((r) => r.stationId === this.targetStationId);
			return existing
				? { ...existing }
				: {
						stationId: this.targetStationId,
						upstreamUrl: "",
					};
		},
		// merge the edited rule back into the complete set that gets persisted
		transformWriteValues(rule: OcppForwarderRule): OcppForwarderRule[] {
			const list = this.rules.map((r) => ({ ...r }));
			const index = list.findIndex((r) => r.stationId === rule.stationId);
			if (index >= 0) {
				list[index] = rule;
			} else {
				list.push(rule);
			}
			return list;
		},
		async removeRule() {
			this.removing = true;
			try {
				const list = this.rules.filter((r) => r.stationId !== this.targetStationId);
				const res = await api.post("/config/ocppforwarder", list, {
					validateStatus: (code: number) => [200, 202, 400].includes(code),
				});
				if (res.status === 200 || res.status === 202) {
					this.$emit("changed");
					await closeModal();
				}
			} catch (e) {
				console.error(e);
			}
			this.removing = false;
		},
	},
});
</script>
