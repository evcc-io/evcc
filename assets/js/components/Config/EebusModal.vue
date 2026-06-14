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
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import type { EebusConfig, EebusStatus, YamlSource } from "@/types/evcc";
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
	computed: {
		fromYaml(): boolean {
			return this.yamlSource === "file";
		},
	},
	methods: {
		formId(s: string) {
			return `eebus-${s}`;
		},
	},
};
</script>
