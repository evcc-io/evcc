<template>
	<JsonModal
		id="eebusModal"
		:title="$t('config.eebus.title')"
		:description="$t('config.eebus.description')"
		docs="/docs/reference/configuration/eebus"
		endpoint="/config/eebus"
		state-key="eebus"
		data-testid="eebus-modal"
		@changed="$emit('changed')"
	>
		<template #default="{ values }: { values: Eebus }">
			<FormRow
				:id="formId('shipid')"
				:label="$t('config.eebus.shipid')"
				:help="$t('config.eebus.shipidHelp')"
				optional
			>
				<PropertyField :id="formId('shipid')" v-model="values.shipid" type="String" />
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
					property="interfaces"
					type="List"
				/>
			</FormRow>

			<h6>{{ $t("config.eebus.https") }}</h6>
			<FormRow
				:id="formId('certificate-public')"
				:label="$t('config.eebus.certificate.public')"
			>
				<PropertyCertField
					:id="formId('certificate-public')"
					:model-value="values.certificate?.public"
				/>
			</FormRow>
			<FormRow
				:id="formId('certificate-private')"
				:label="$t('config.eebus.certificate.private')"
			>
				<PropertyCertField
					:id="formId('certificate-private')"
					:model-value="values.certificate?.private"
				/>
			</FormRow>
		</template>
	</JsonModal>
</template>

<script lang="ts">
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import type { Eebus } from "@/types/evcc";
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import PropertyCertField from "./PropertyCertField.vue";

export default {
	name: "EebusModal",
	components: { JsonModal, FormRow, PropertyField, PropertyCertField },
	emits: ["changed"],
	methods: {
		formId(s: string) {
			return `eebus-${s}`;
		},
	},
};
</script>
