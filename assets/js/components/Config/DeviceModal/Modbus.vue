<template>
	<FormRow
		v-if="showConnectionOptions"
		id="modbusTcpIp"
		:label="$t('config.modbus.connection')"
		:help="
			connectionData === MODBUS_CONNECTION.TCPIP
				? $t('config.modbus.connectionHintTcpip')
				: $t('config.modbus.connectionHintSerial')
		"
	>
		<div class="btn-group" role="group">
			<input
				:id="formId('modbusTcpIp')"
				v-model="connectionData"
				type="radio"
				class="btn-check"
				:name="formId('modbusConnection')"
				value="tcpip"
				tabindex="0"
				autocomplete="off"
			/>
			<label class="btn btn-outline-primary" :for="formId('modbusTcpIp')">
				{{ $t("config.modbus.connectionValueTcpip") }}
			</label>
			<input
				:id="formId('modbusSerial')"
				v-model="connectionData"
				type="radio"
				class="btn-check"
				:name="formId('modbusConnection')"
				value="serial"
				tabindex="0"
				autocomplete="off"
			/>
			<label class="btn btn-outline-primary" :for="formId('modbusSerial')">
				{{ $t("config.modbus.connectionValueSerial") }}
			</label>
		</div>
	</FormRow>
	<FormRow v-if="!hideModbusId" id="modbusId" :label="$t('config.modbus.id')">
		<PropertyField
			id="modbusId"
			property="id"
			type="Int"
			class="me-2"
			required
			:model-value="id || defaultId || 1"
			@change="$emit('update:id', $event.target.value)"
		/>
	</FormRow>
	<div v-if="connectionData === MODBUS_CONNECTION.TCPIP">
		<FormRow
			id="modbusHost"
			:label="$t('config.modbus.host')"
			:help="$t('config.modbus.hostHint')"
		>
			<PropertyField
				id="modbusHost"
				property="host"
				type="String"
				class="me-2"
				required
				:model-value="host"
				@change="$emit('update:host', $event.target.value)"
			/>
		</FormRow>
		<FormRow id="modbusPort" :label="$t('config.modbus.port')">
			<PropertyField
				id="modbusPort"
				property="port"
				type="Int"
				class="me-2 w-50"
				required
				:model-value="port || defaultPort || 502"
				@change="$emit('update:port', $event.target.value)"
			/>
		</FormRow>
		<FormRow
			v-if="showProtocolOptions"
			id="modbusTcp"
			:label="$t('config.modbus.protocol')"
			:help="
				protocolData === 'tcp'
					? $t('config.modbus.protocolHintTcp')
					: $t('config.modbus.protocolHintRtu')
			"
		>
			<div class="btn-group" role="group">
				<input
					:id="formId('modbusTcp')"
					v-model="protocolData"
					type="radio"
					class="btn-check"
					:name="formId('modbusProtocol')"
					value="tcp"
					tabindex="0"
					autocomplete="off"
				/>
				<label class="btn btn-outline-primary" :for="formId('modbusTcp')">
					{{ $t("config.modbus.protocolValueTcp") }}
				</label>
				<input
					:id="formId('modbusRtu')"
					v-model="protocolData"
					type="radio"
					class="btn-check"
					:name="formId('modbusProtocol')"
					value="rtu"
					tabindex="0"
					autocomplete="off"
				/>
				<label class="btn btn-outline-primary" :for="formId('modbusRtu')">
					{{ $t("config.modbus.protocolValueRtu") }}
				</label>
			</div>
		</FormRow>
	</div>
	<div v-else>
		<FormRow
			id="modbusDevice"
			:label="$t('config.modbus.device')"
			:help="$t('config.modbus.deviceHint')"
		>
			<PropertyField
				id="modbusDevice"
				property="device"
				type="String"
				class="me-2"
				required
				:model-value="device"
				@change="$emit('update:device', $event.target.value)"
			/>
		</FormRow>
		<FormRow id="modbusBaudrate" :label="$t('config.modbus.baudrate')">
			<PropertyField
				id="modbusBaudrate"
				property="baudrate"
				type="Choice"
				class="me-2 w-50"
				:choice="baudrateOptions"
				required
				:model-value="baudrate || defaultBaudrate"
				@change="$emit('update:baudrate', parseInt($event.target.value))"
			/>
		</FormRow>
		<FormRow id="modbusComset" :label="$t('config.modbus.comset')">
			<PropertyField
				id="modbusComset"
				property="comset"
				type="Choice"
				class="me-2 w-50"
				:choice="comsetOptions"
				required
				:model-value="comset || defaultComset || '8N1'"
				@change="$emit('update:comset', $event.target.value)"
			/>
		</FormRow>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import FormRow from "../FormRow.vue";
import PropertyField from "../PropertyField.vue";
import type { PropType } from "vue";
import type { ModbusCapability } from "./index";
import { MODBUS_BAUDRATE, MODBUS_COMSET, MODBUS_CONNECTION, MODBUS_PROTOCOL } from "@/types/evcc";
type Modbus = "rs485serial" | "rs485tcpip" | "tcpip";

export default defineComponent({
	name: "Modbus",
	components: { FormRow, PropertyField },
	props: {
		capabilities: {
			type: Array as PropType<ModbusCapability[]>,
			default: () => [],
		},
		modbus: String as PropType<Modbus>,
		host: String,
		port: [Number, String],
		id: [Number, String],
		baudrate: [Number, String],
		comset: String,
		device: String,
		connection: {
			type: String as PropType<MODBUS_CONNECTION>,
			default: () => MODBUS_CONNECTION.TCPIP,
		},
		protocol: {
			type: String as PropType<MODBUS_PROTOCOL>,
			default: () => MODBUS_PROTOCOL.TCP,
		},
		defaultPort: Number,
		defaultId: Number,
		defaultComset: String,
		defaultBaudrate: Number,
		hideModbusId: Boolean,
	},
	emits: [
		"update:modbus",
		"update:id",
		"update:host",
		"update:port",
		"update:device",
		"update:baudrate",
		"update:comset",
		"update:protocol",
	],
	data() {
		return {
			connectionData: this.connection as MODBUS_CONNECTION,
			protocolData: this.protocol as MODBUS_PROTOCOL,
			MODBUS_PROTOCOL,
			MODBUS_CONNECTION,
		};
	},
	computed: {
		selectedModbus(): Modbus {
			if (this.connectionData === MODBUS_CONNECTION.SERIAL) {
				return "rs485serial";
			}
			return this.protocolData === "rtu" ? "rs485tcpip" : "tcpip";
		},
		showConnectionOptions() {
			return this.capabilities.includes("rs485");
		},
		showProtocolOptions() {
			return (
				this.connectionData === MODBUS_CONNECTION.TCPIP &&
				this.capabilities.includes("rs485")
			);
		},
		comsetOptions() {
			return Object.values(MODBUS_COMSET).map((v) => {
				return { key: v, name: v };
			});
		},
		baudrateOptions() {
			return Object.values(MODBUS_BAUDRATE)
				.filter((v) => typeof v === "number")
				.map((v) => {
					return { key: v, name: `${v}` };
				});
		},
	},
	watch: {
		selectedModbus(newValue: Modbus) {
			this.$emit("update:modbus", newValue);
		},
		options(newValue: ModbusCapability[]) {
			this.setProtocolByCapabilities(newValue);
			this.$emit("update:modbus", this.selectedModbus);
		},
		modbus(newValue: Modbus) {
			if (newValue) {
				this.setConnectionAndProtocolByModbus(newValue);
			}
		},
		protocolData(newProtocol) {
			this.$emit("update:protocol", newProtocol);
		},
	},
	mounted() {
		this.setConnectionAndProtocolByModbus(this.modbus);
		this.$emit("update:modbus", this.selectedModbus);
	},
	methods: {
		setProtocolByCapabilities(capabilities: ModbusCapability[]) {
			this.protocolData = capabilities.includes("tcpip")
				? MODBUS_PROTOCOL.TCP
				: MODBUS_PROTOCOL.RTU;
		},
		setConnectionAndProtocolByModbus(modbus?: Modbus) {
			switch (modbus) {
				case "rs485serial":
					this.connectionData = MODBUS_CONNECTION.SERIAL;
					this.protocolData = MODBUS_PROTOCOL.RTU;
					break;
				case "rs485tcpip":
					this.connectionData = MODBUS_CONNECTION.TCPIP;
					this.protocolData = MODBUS_PROTOCOL.RTU;
					break;
				case "tcpip":
					this.connectionData = MODBUS_CONNECTION.TCPIP;
					this.protocolData = MODBUS_PROTOCOL.RTU;
					break;
			}
		},
		formId(name: string): string {
			return `${name}-${this.id}`;
		},
	},
});
</script>
