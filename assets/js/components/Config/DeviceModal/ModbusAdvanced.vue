<template>
	<FormRow
		:id="formId('modbusTimeout')"
		:label="$t('config.modbus.timeout')"
		:help="helpOverrides['timeout'] || $t('config.modbus.timeoutHint')"
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
		:help="helpOverrides['delay'] || $t('config.modbus.delayHint')"
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
import type { PropType } from "vue";
import FormRow from "../FormRow.vue";
import PropertyField from "../PropertyField.vue";

export default defineComponent({
	name: "ModbusAdvanced",
	components: { FormRow, PropertyField },
	props: {
		componentId: { type: String, required: true },
		delay: [Number, String],
		timeout: [Number, String],
		// per-field help texts a template overrides, keyed by modbus param name
		helpOverrides: { type: Object as PropType<Record<string, string>>, default: () => ({}) },
	},
	emits: ["update:delay", "update:timeout"],
	methods: {
		formId(name: string): string {
			return `${name}-${this.componentId}`;
		},
	},
});
</script>
