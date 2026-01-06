<template>
	<FormRow
		v-if="showConnectionOptions"
		id="modbusTcpIp"
		:label="$t('config.modbus.connection')"
		:help="
			connection === MODBUS_CONNECTION.TCPIP
				? $t('config.modbus.connectionHintTcpip')
				: $t('config.modbus.connectionHintSerial')
		"
	>
		<div class="btn-group" role="group">
			<input
				:id="formId('modbusTcpIp')"
				v-model="connection"
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
				v-model="connection"
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
	<div v-if="connection === MODBUS_CONNECTION.TCPIP">
		<FormRow
			:id="formId('modbusHost')"
			:label="$t('config.modbus.host')"
			:help="$t('config.modbus.hostHint')"
		>
			<PropertyField
				:id="formId('modbusHost')"
				property="host"
				type="String"
				class="me-2"
				required
				:model-value="host"
				@change="$emit('update:host', $event.target.value)"
			/>
		</FormRow>
		<FormRow :id="formId('modbusPort')" :label="$t('config.modbus.port')">
			<PropertyField
				:id="formId('modbusPort')"
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
			:id="formId('modbusTcp')"
			:label="$t('config.modbus.protocol')"
			:help="
				protocol === 'tcp'
					? $t('config.modbus.protocolHintTcp')
					: $t('config.modbus.protocolHintRtu')
			"
		>
			<div class="btn-group" role="group">
				<input
					:id="formId('modbusTcp')"
					v-model="protocol"
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
					v-model="protocol"
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
			:id="formId('modbusDevice')"
			:label="$t('config.modbus.device')"
			:help="$t('config.modbus.deviceHint')"
			data-testid="modbus-device"
		>
			<PropertyField
				:id="formId('modbusDevice')"
				v-model:model-value="deviceModel"
				property="device"
				type="String"
				class="me-2"
				required
				:service-values="deviceServiceValues"
			/>
		</FormRow>
		<FormRow :id="formId('modbusBaudrate')" :label="$t('config.modbus.baudrate')">
			<PropertyField
				:id="formId('modbusBaudrate')"
				property="baudrate"
				type="Choice"
				class="me-2 w-50"
				:choice="baudrateOptions"
				required
				:model-value="baudrate || defaultBaudrate"
				@change="$emit('update:baudrate', parseInt($event.target.value))"
			/>
		</FormRow>
		<FormRow :id="formId('modbusComset')" :label="$t('config.modbus.comset')">
			<PropertyField
				:id="formId('modbusComset')"
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
import { loadServiceValues } from "./index";
import {
	MODBUS_BAUDRATE,
	MODBUS_COMSET,
	MODBUS_CONNECTION,
	MODBUS_PROTOCOL,
	MODBUS_TYPE,
} from "@/types/evcc";

export default defineComponent({
	name: "Modbus",
	components: { FormRow, PropertyField },
	props: {
		componentId: { type: String, required: true },
		capabilities: {
			type: Array as PropType<ModbusCapability[]>,
			default: () => [],
		},
		modbus: String as PropType<MODBUS_TYPE>,
		host: String,
		port: [Number, String],
		id: [Number, String],
		baudrate: [Number, String],
		comset: String,
		device: String,
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
	],
	data() {
		return {
			connection: MODBUS_CONNECTION.TCPIP as MODBUS_CONNECTION,
			protocol: MODBUS_PROTOCOL.TCP as MODBUS_PROTOCOL,
			MODBUS_PROTOCOL,
			MODBUS_CONNECTION,
			deviceServiceValues: [] as string[],
			localDevice: undefined as string | undefined,
		};
	},
	computed: {
		deviceModel: {
			get(): string | undefined {
				return this.localDevice !== undefined ? this.localDevice : this.device;
			},
			set(value: string | undefined) {
				this.localDevice = value;
				this.$emit("update:device", value);
			},
		},
		selectedModbus(): MODBUS_TYPE {
			if (this.connection === MODBUS_CONNECTION.SERIAL) {
				return MODBUS_TYPE.RS485_SERIAL;
			}
			return this.protocol === MODBUS_PROTOCOL.RTU
				? MODBUS_TYPE.RS485_TCPIP
				: MODBUS_TYPE.TCPIP;
		},
		showConnectionOptions() {
			return this.capabilities.includes("rs485");
		},
		showProtocolOptions() {
			return (
				this.connection === MODBUS_CONNECTION.TCPIP && this.capabilities.includes("rs485")
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
		selectedModbus(newValue: MODBUS_TYPE) {
			this.$emit("update:modbus", newValue);
		},
		options(newValue: ModbusCapability[]) {
			this.setProtocolByCapabilities(newValue);
			this.$emit("update:modbus", this.selectedModbus);
		},
		modbus(newValue: MODBUS_TYPE) {
			if (newValue) {
				this.setConnectionAndProtocolByModbus(newValue);
			}
		},
		connection() {
			this.applyServiceDefault();
		},
		device(newValue: string | undefined) {
			// Sync prop to local state
			if (newValue !== this.localDevice) {
				this.localDevice = newValue;
			}
		},
	},
	mounted() {
		this.localDevice = this.device;
		this.setConnectionAndProtocolByModbus(this.modbus);
		this.$emit("update:modbus", this.selectedModbus);
		this.updateServiceValues();
	},
	methods: {
		setProtocolByCapabilities(capabilities: ModbusCapability[]) {
			this.protocol = capabilities.includes("tcpip")
				? MODBUS_PROTOCOL.TCP
				: MODBUS_PROTOCOL.RTU;
		},
		setConnectionAndProtocolByModbus(modbus?: MODBUS_TYPE) {
			switch (modbus) {
				case MODBUS_TYPE.RS485_SERIAL:
					this.connection = MODBUS_CONNECTION.SERIAL;
					this.protocol = MODBUS_PROTOCOL.RTU;
					break;
				case MODBUS_TYPE.RS485_TCPIP:
					this.connection = MODBUS_CONNECTION.TCPIP;
					this.protocol = MODBUS_PROTOCOL.RTU;
					break;
				case MODBUS_TYPE.TCPIP:
					this.connection = MODBUS_CONNECTION.TCPIP;
					this.protocol = MODBUS_PROTOCOL.TCP;
					break;
			}
		},
		formId(name: string): string {
			return `${name}-${this.componentId}`;
		},
		async updateServiceValues() {
			this.deviceServiceValues = await loadServiceValues("hardware/serial");
			this.applyServiceDefault();
		},
		applyServiceDefault() {
			// auto-apply device value if it's needed and exactly one option exists
			if (
				this.connection === MODBUS_CONNECTION.SERIAL &&
				this.deviceServiceValues.length === 1 &&
				!this.device
			) {
				this.deviceModel = this.deviceServiceValues[0];
			}
		},
	},
});
</script>
