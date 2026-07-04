<template>
	<JsonModal
		name="eebus"
		:title="$t('config.eebus.title')"
		:description="$t('config.eebus.description')"
		docs="/docs/reference/configuration/eebus"
		endpoint="/config/eebus"
		state-key="eebus.config"
		:no-buttons="fromYaml"
		:confirm-remove="$t('config.eebus.removeConfirm')"
		@changed="$emit('changed')"
		@open="loadPairings"
	>
		<template #default="{ values }: { values: EebusConfig }">
			<p v-if="fromYaml" class="text-muted">
				{{ $t("config.general.fromYamlHint") }}
			</p>
			<FormRow
				v-if="values.shipid"
				:id="formId('shipid-display')"
				:label="$t('config.eebus.shipid')"
				:help="$t('config.eebus.shipidExplain')"
			>
				<input
					:id="formId('shipid-display')"
					:value="values.shipid"
					readonly
					class="form-control text-muted"
				/>
			</FormRow>
			<FormRow
				v-if="status.ski"
				:id="formId('ski-display')"
				:label="$t('config.eebus.ski')"
				:help="$t('config.eebus.skiExplain')"
			>
				<input
					:id="formId('ski-display')"
					:value="status.ski"
					readonly
					class="form-control text-muted"
				/>
			</FormRow>
			<FormRow
				v-if="qrDataUrl"
				:id="formId('qr-display')"
				:label="$t('config.eebus.qr')"
				:help="$t('config.eebus.qrExplain')"
			>
				<img :src="qrDataUrl" :alt="$t('config.eebus.qr')" class="qr-code" />
			</FormRow>
			<div v-if="status.ski" class="mb-4">
				<h6 class="mb-3">{{ $t("config.eebus.pairings") }}</h6>
				<div
					v-for="pairing in pairedDevices"
					:key="pairing.shipID || pairing.ski"
					data-testid="eebus-pairing"
					class="mb-2"
				>
					<div
						class="d-flex align-items-center justify-content-between py-2 ps-3 pe-2 border rounded"
					>
						<div class="flex-grow-1 fw-semibold text-truncate">
							{{ pairing.shipID || pairing.ski }}
						</div>
						<small
							v-if="pairing.shipID && pairing.ski"
							class="text-muted ms-2 me-2 text-truncate"
						>
							{{ pairing.ski }}
						</small>
						<button
							type="button"
							class="btn btn-sm btn-outline-secondary border-0"
							:aria-label="$t('config.eebus.removePairing')"
							@click="removePairing(pairing)"
						>
							<shopicon-regular-trash
								size="s"
								class="flex-shrink-0"
							></shopicon-regular-trash>
						</button>
					</div>
				</div>
				<div v-if="!pairedDevices.length" class="text-muted small mb-2">
					{{ $t("config.eebus.noPairings") }}
				</div>
			</div>
			<div v-if="status.ski" class="mb-4">
				<h6 class="mb-3">{{ $t("config.eebus.configuredDevices") }}</h6>
				<div
					v-for="pairing in configuredDevices"
					:key="pairing.ski"
					data-testid="eebus-configured-device"
					class="mb-2"
				>
					<div
						class="d-flex align-items-center justify-content-between py-2 ps-3 pe-2 border rounded"
					>
						<div class="flex-grow-1 fw-semibold text-truncate">
							{{ pairing.ski }}
						</div>
					</div>
				</div>
				<div v-if="!configuredDevices.length" class="text-muted small mb-2">
					{{ $t("config.eebus.noConfiguredDevices") }}
				</div>
			</div>
			<PropertyCollapsible v-if="!fromYaml">
				<template #advanced>
					<div class="alert alert-danger">
						{{ $t("config.eebus.descriptionAdvanced") }}
					</div>
					<FormRow
						:id="formId('shipid')"
						:label="$t('config.eebus.shipid')"
						:help="$t('config.eebus.shipidHelp')"
					>
						<PropertyField
							:id="formId('shipid')"
							v-model="values.shipid"
							type="String"
							required
						/>
					</FormRow>
					<FormRow
						:id="formId('port')"
						:label="$t('config.eebus.port')"
						:help="$t('config.eebus.portHelp')"
						optional
					>
						<PropertyField
							:id="formId('port')"
							v-model="values.port"
							property="port"
							type="Int"
						/>
					</FormRow>
					<FormRow
						:id="formId('interfaces')"
						:label="$t('config.eebus.interfaces')"
						:help="$t('config.eebus.interfacesHelp')"
						optional
						example="eth0"
					>
						<PropertyField
							:id="formId('interfaces')"
							v-model="values.interfaces"
							type="List"
						/>
					</FormRow>
					<h6>{{ $t("config.eebus.certificate.title") }}</h6>
					<FormRow
						:id="formId('certificate-public')"
						:label="$t('config.eebus.certificate.public')"
					>
						<PropertyCertField
							:id="formId('certificate-public')"
							:model-value="values.certificate?.public"
							required
							@update:model-value="
								values.certificate ? (values.certificate.public = $event) : ''
							"
						/>
					</FormRow>
					<FormRow
						:id="formId('certificate-private')"
						:label="$t('config.eebus.certificate.private')"
					>
						<PropertyCertField
							:id="formId('certificate-private')"
							:model-value="values.certificate?.private"
							required
							@update:model-value="
								values.certificate ? (values.certificate.private = $event) : ''
							"
						/>
					</FormRow>
				</template>
			</PropertyCollapsible>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import type { PropType } from "vue";
import QRCode from "qrcode";
import "@h2d2/shopicons/es/regular/trash";
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import type { EebusConfig, EebusPairing, EebusStatus, YamlSource } from "@/types/evcc";
import api from "@/api";
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import PropertyCertField from "./PropertyCertField.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";

export default {
	name: "EebusModal",
	components: {
		JsonModal,
		FormRow,
		PropertyField,
		PropertyCertField,
		PropertyCollapsible,
	},
	props: {
		status: {
			type: Object as PropType<EebusStatus>,
			default: () => ({}),
		},
		yamlSource: String as PropType<YamlSource>,
	},
	emits: ["changed"],
	data() {
		return {
			qrDataUrl: null as string | null,
			pairings: [] as EebusPairing[],
		};
	},
	computed: {
		fromYaml(): boolean {
			return this.yamlSource === "file";
		},
		pairedDevices(): EebusPairing[] {
			return this.pairings.filter((p) => p.source === "paired");
		},
		configuredDevices(): EebusPairing[] {
			return this.pairings.filter((p) => p.source === "ski");
		},
	},
	watch: {
		"status.qr": {
			immediate: true,
			async handler(qr?: string) {
				if (!qr) {
					this.qrDataUrl = null;
					return;
				}
				try {
					this.qrDataUrl = await QRCode.toDataURL(qr, { width: 200, margin: 1 });
				} catch {
					this.qrDataUrl = null;
				}
			},
		},
	},
	methods: {
		formId(s: string) {
			return `eebus-${s}`;
		},
		async loadPairings() {
			try {
				const res = await api.get("config/service/eebus/pairings");
				this.pairings = res.data || [];
			} catch {
				this.pairings = [];
			}
		},
		async removePairing(pairing: EebusPairing) {
			if (!window.confirm(this.$t("config.eebus.removePairingConfirm"))) {
				return;
			}
			try {
				const id = encodeURIComponent(pairing.shipID || pairing.ski);
				await api.delete(`config/service/eebus/pairings/${id}`);
			} finally {
				await this.loadPairings();
			}
		},
	},
};
</script>

<style scoped>
.qr-code {
	image-rendering: pixelated;
}
</style>
