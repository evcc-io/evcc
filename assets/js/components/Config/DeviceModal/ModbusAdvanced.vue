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
			class="me-2"
			:model-value="delay"
			@update:model-value="(v) => $emit('update:delay', v)"
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
	},
	emits: ["update:delay", "update:timeout"],
	methods: {
		formId(name: string): string {
			return `${name}-${this.componentId}`;
		},
	},
});
</script>
