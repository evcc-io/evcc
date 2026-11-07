<template>
	<FormRow
		:id="formId('modbusTimeout')"
		:label="$t('config.modbus.timeout')"
		:help="$t('config.modbus.timeoutHint')"
	>
		<PropertyField
			:id="formId('modbusTimeout')"
			property="timeout"
			type="Duration"
			:label="$t('config.modbus.timeout')"
			class="me-2"
			:model-value="timeout"
			@update:model-value="(v) => $emit('update:timeout', v)"
		/>
	</FormRow>
	<FormRow
		:id="formId('modbusDelay')"
		:label="$t('config.modbus.delay')"
		:help="$t('config.modbus.delayHint')"
	>
		<PropertyField
			:id="formId('modbusDelay')"
			property="delay"
			type="Duration"
			:label="$t('config.modbus.delay')"
			class="me-2"
			:model-value="delay"
			@update:model-value="(v) => $emit('update:delay', v)"
		/>
	</FormRow>
	<FormRow
		v-if="showTls"
		:id="formId('modbusTls')"
		:label="$t('config.modbus.tls')"
		:help="$t('config.modbus.tlsHint')"
	>
		<PropertyField
			:id="formId('modbusTls')"
			property="tls"
			type="Bool"
			:label="$t('config.modbus.tls')"
			class="me-2"
			:model-value="tls"
			@update:model-value="(v) => $emit('update:tls', v)"
		/>
	</FormRow>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import FormRow from "../FormRow.vue";
import PropertyField from "../PropertyField.vue";

export default defineComponent({
	name: "ModbusAdvanced",
	components: { FormRow, PropertyField },
	props: {
		componentId: { type: String, required: true },
		delay: [Number, String],
		timeout: [Number, String],
		tls: Boolean,
		showTls: Boolean,
	},
	emits: ["update:delay", "update:timeout", "update:tls"],
	methods: {
		formId(name: string): string {
			return `${name}-${this.componentId}`;
		},
	},
});
</script>
