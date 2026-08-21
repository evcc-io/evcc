<template>
	<JsonModal
		name="ocppreport"
		:title="$t('config.ocppreport.editTitle')"
		:description="$t('config.ocppreport.description')"
		endpoint="/config/ocppreport"
		state-key="ocppreport.config"
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
				values: OcppReportRule;
				changes: boolean;
				save: (close?: boolean) => void;
			}"
		>
			<p v-if="ruleExists" class="mb-3" data-testid="ocppreport-status">
				{{ $t("config.ocppreport.status") }}:
				<span :class="connectionConnected ? 'text-success' : 'text-danger'">{{
					connectionLabel
				}}</span>
			</p>
			<div
				v-if="sessionError"
				class="alert alert-danger"
				role="alert"
				data-testid="ocppreport-error"
			>
				{{ sessionError }}
			</div>
			<FormRow
				id="ocppreportUpstreamUrl"
				:label="$t('config.ocppreport.upstreamUrl')"
				:help="$t('config.ocppreport.upstreamUrlHelp')"
				example="wss://billing.example.com/ocpp"
			>
				<input
					id="ocppreportUpstreamUrl"
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
				id="ocppreportUsername"
				:label="$t('config.ocppreport.username')"
				:help="$t('config.ocppreport.usernameHelp')"
				optional
			>
				<input
					id="ocppreportUsername"
					v-model="values.username"
					type="text"
					class="form-control"
					spellcheck="false"
					autocomplete="off"
				/>
			</FormRow>
			<FormRow
				id="ocppreportPassword"
				:label="$t('config.ocppreport.password')"
				:help="$t('config.ocppreport.passwordHelp')"
				optional
			>
				<input
					id="ocppreportPassword"
					v-model="values.password"
					type="password"
					class="form-control"
					autocomplete="new-password"
				/>
			</FormRow>
			<FormRow
				id="ocppreportStationId"
				:label="$t('config.ocppreport.stationId')"
				:help="$t('config.ocppreport.stationIdHelp')"
				optional
			>
				<input
					id="ocppreportStationId"
					v-model="values.stationId"
					type="text"
					class="form-control"
					:placeholder="defaultStationId"
					spellcheck="false"
					autocomplete="off"
				/>
			</FormRow>
			<FormRow
				id="ocppreportIdTag"
				:label="$t('config.ocppreport.idTag')"
				:help="$t('config.ocppreport.idTagHelp')"
				optional
			>
				<input
					id="ocppreportIdTag"
					v-model="values.idTag"
					type="text"
					class="form-control"
					placeholder="EVCC"
					spellcheck="false"
					autocomplete="off"
				/>
			</FormRow>
			<PropertyCollapsible>
				<template #advanced>
					<FormRow id="ocppreportInsecure" :label="$t('config.ocppreport.labelInsecure')">
						<div class="d-flex">
							<input
								id="ocppreportInsecure"
								v-model="values.insecure"
								class="form-check-input"
								type="checkbox"
							/>
							<label class="form-check-label ms-2" for="ocppreportInsecure">
								{{ $t("config.ocppreport.labelCheckInsecure") }}
							</label>
						</div>
					</FormRow>
					<FormRow id="ocppreportCaCert" :label="$t('config.ocppreport.labelCaCert')" optional>
						<PropertyCertField id="ocppreportCaCert" v-model="values.caCert" />
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
import type { OcppReportRule, OcppReportSession } from "@/types/evcc";
import { getModal, closeModal } from "@/configModal";
import api from "@/api";
import store from "@/store";

export default defineComponent({
	name: "OcppReportModal",
	components: { JsonModal, FormRow, PropertyCollapsible, PropertyCertField },
	emits: ["changed"],
	data() {
		return { removing: false };
	},
	computed: {
		// loadpoint title the modal is editing, carried via the config modal stack
		targetLoadpointTitle(): string {
			return getModal("ocppreport")?.loadpoint || "";
		},
		defaultStationId(): string {
			return `evcc-${this.targetLoadpointTitle}`;
		},
		rules(): OcppReportRule[] {
			return store.state?.ocppreport?.config || [];
		},
		ruleExists(): boolean {
			return this.rules.some((r) => r.loadpointTitle === this.targetLoadpointTitle);
		},
		session(): OcppReportSession | undefined {
			return store.state?.ocppreport?.status?.find(
				(s) => s.loadpointTitle === this.targetLoadpointTitle
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
		// pick the rule for the target loadpoint, or seed a new one prefilled with the title
		transformReadValues(rules: OcppReportRule[]): OcppReportRule {
			const list = Array.isArray(rules) ? rules : [];
			const existing = list.find((r) => r.loadpointTitle === this.targetLoadpointTitle);
			return existing
				? { ...existing }
				: {
						loadpointTitle: this.targetLoadpointTitle,
						upstreamUrl: "",
					};
		},
		// merge the edited rule back into the complete set that gets persisted
		transformWriteValues(rule: OcppReportRule): OcppReportRule[] {
			const list = this.rules.map((r) => ({ ...r }));
			const index = list.findIndex((r) => r.loadpointTitle === rule.loadpointTitle);
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
				const list = this.rules.filter((r) => r.loadpointTitle !== this.targetLoadpointTitle);
				const res = await api.post("/config/ocppreport", list, {
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
