<template>
	<Modbus
		v-model:baudrate="localConnection.settings.baudrate"
		v-model:comset="localConnection.settings.comset"
		v-model:device="localConnection.settings.device"
		:component-id="`proxy-${index}`"
		:host="getHost(localConnection.settings.uri)"
		:port="getPort(localConnection.settings.uri)"
		:capabilities="['rs485', 'tcpip']"
		hide-modbus-id
		:default-baudrate="1200"
		:default-comset="'8N1'"
		:default-port="502"
		:modbus="initialModbus"
		@update:host="(host) => updateHost(host)"
		@update:port="(port) => updatePort(port)"
		@update:modbus="(modbus) => updateModbus(modbus)"
	/>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import Modbus from "./DeviceModal/Modbus.vue";
import { MODBUS_TYPE, type ModbusProxy, type ModbusProxySettings } from "@/types/evcc";
import deepClone from "@/utils/deepClone";

export default defineComponent({
	name: "ModbusProxyConnection",
	components: { Modbus },
	props: {
		connection: {
			type: Object as () => ModbusProxy,
			required: true,
		},
		index: {
			type: Number,
			required: true,
		},
	},
	emits: ["update:connection"],
	data() {
		return {
			localConnection: deepClone(this.connection),
			initialModbus: undefined as MODBUS_TYPE | undefined,
		};
	},
	watch: {
		localConnection: {
			handler(newVal: ModbusProxy) {
				if (newVal) {
					this.$emit("update:connection", newVal);
				}
			},
			deep: true,
		},
	},
	mounted() {
		this.initialModbus = this.getModbus(this.localConnection.settings);
	},
	methods: {
		getModbus(s: ModbusProxySettings) {
			if (this.initialModbus) {
				return this.initialModbus;
			}

			if (s.device) {
				return MODBUS_TYPE.RS485_SERIAL;
			}
			return s.rtu ? MODBUS_TYPE.RS485_TCPIP : MODBUS_TYPE.TCPIP;
		},
		getHost(uri?: string) {
			return uri?.split(":")[0] || "";
		},
		getPort(uri?: string) {
			return uri?.split(":")[1] || "";
		},
		updateHost(newHost?: string) {
			const port = this.getPort(this.localConnection.settings.uri);

			if (port === "" && newHost === undefined) {
				this.localConnection.settings.uri = undefined;
			} else {
				this.localConnection.settings.uri = `${newHost === undefined ? "" : newHost}:${port}`;
			}
		},
		updatePort(newPort?: string) {
			const host = this.getHost(this.localConnection.settings.uri);
			if (host === "" && newPort === undefined) {
				this.localConnection.settings.uri = undefined;
			} else {
				this.localConnection.settings.uri = `${host}:${newPort === undefined ? "" : newPort}`;
			}
		},
		updateModbus(modbus: MODBUS_TYPE) {
			this.initialModbus = modbus;

			switch (modbus) {
				case MODBUS_TYPE.RS485_SERIAL:
					this.localConnection.settings.uri = undefined;
					this.localConnection.settings.rtu = undefined;
					break;
				case MODBUS_TYPE.RS485_TCPIP:
				case MODBUS_TYPE.TCPIP:
					this.localConnection.settings.device = undefined;
					this.localConnection.settings.baudrate = undefined;
					this.localConnection.settings.comset = undefined;
					this.localConnection.settings.rtu = modbus === MODBUS_TYPE.RS485_TCPIP;
					break;
			}
		},
	},
});
</script>
