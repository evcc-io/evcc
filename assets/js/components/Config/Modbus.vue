<template>
	<FormRow id="modbus" label="Modbus" help="TODO">
		<PropertyField
			id="modbus"
			property="modbus"
			:validValues="['rs485serial', 'rs485tcpip', 'tcpip']"
			class="me-2"
			required
			@change="$emit('update:modbus', $event.target.value)"
		/>
	</FormRow>
	<FormRow id="modbusId" label="ID" help="TODO">
		<PropertyField
			id="modbusId"
			property="id"
			type="Number"
			class="me-2"
			required
			@change="$emit('update:id', $event.target.value)"
		/>
	</FormRow>
	<FormRow id="modbusHost" label="Host" help="TODO">
		<PropertyField
			id="modbusHost"
			property="host"
			type="String"
			class="me-2"
			required
			@change="$emit('update:host', $event.target.value)"
		/>
	</FormRow>
	<FormRow id="modbusPort" label="Port" help="TODO">
		<PropertyField
			id="modbusPort"
			property="port"
			type="Number"
			class="me-2"
			required
			@change="$emit('update:port', $event.target.value)"
		/>
	</FormRow>
</template>

<script>
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";

export default {
	name: "Modbus",
	components: { FormRow, PropertyField },
	props: {
		choices: Array,
		modbus: Array,
		id: Number,
		host: String,
		port: Number,
	},
	emits: ["update:modbus", "update:id", "update:host", "update:port"],
};
</script>
