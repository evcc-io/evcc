<template>
	<div>
		<FormRow
			v-if="showConnectionOptions"
			id="modbusTcpIp"
			:label="$t('config.modbus.connection')"
			:help="
				connection === 'tcpip'
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
		<FormRow v-if="!isProxy" id="modbusId" :label="$t('config.modbus.id')">
			<PropertyField
				id="modbusId"
				property="id"
				type="Int"
				class="me-2"
				required
				:model-value="id || defaultId"
				@change="$emit('update:id', $event.target.value)"
			/>
		</FormRow>
		<div v-if="connection === 'tcpip'">
			<FormRow id="modbusHost" :label="$t('config.modbus.host')" example="192.0.2.2">
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
					:model-value="port || defaultPort"
					@change="$emit('update:port', $event.target.value)"
				/>
			</FormRow>
			<FormRow
				v-if="showProtocolOptions"
				id="modbusTcp"
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
			<FormRow id="modbusDevice" :label="$t('config.modbus.device')" example="/dev/ttyUSB0">
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
					@change="$emit('update:baudrate', $event.target.value)"
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
					:model-value="comset || defaultComset"
					@change="$emit('update:comset', $event.target.value)"
				/>
			</FormRow>
		</div>
		<FormRow
			v-if="isProxy"
			id="serialConnectionReadonly"
			:label="$t('config.modbus.readonly')"
			:help="
				readonly === MODBUS_PROXY_READONLY.TRUE
					? $t('config.modbus.readonlyHelpSilent')
					: readonly === MODBUS_PROXY_READONLY.FALSE
						? $t('config.modbus.readonlyHelpNo')
						: readonly === MODBUS_PROXY_READONLY.DENY
							? $t('config.modbus.readonlyHelpError')
							: $t('config.modbus.readonlyHelpDefault')
			"
		>
			<SelectGroup
				id="serialConnectionReadonly"
				:model-value="readonly || defaultReadonly"
				class="w-100"
				:options="readonlyOptions"
				transparent
				@update:model-value="$emit('update:readonly', $event)"
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
import { MODBUS_BAUDRATE, MODBUS_COMSET, MODBUS_PROXY_READONLY } from "@/types/evcc";
import SelectGroup from "@/components/Helper/SelectGroup.vue";
type Modbus = "rs485serial" | "rs485tcpip" | "tcpip";
type ConnectionOption = "tcpip" | "serial";
type ProtocolOption = "tcp" | "rtu";

export default defineComponent({
	name: "Modbus",
	components: { FormRow, PropertyField, SelectGroup },
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
		readonly: String as PropType<MODBUS_PROXY_READONLY>,
		defaultPort: { type: Number, default: 502 },
		defaultId: { type: Number, default: 1 },
		defaultComset: { type: String as PropType<MODBUS_COMSET>, default: "8N1" },
		defaultBaudrate: { type: Number as PropType<MODBUS_BAUDRATE>, default: 1200 },
		defaultReadonly: {
			type: String as PropType<MODBUS_PROXY_READONLY>,
			default: MODBUS_PROXY_READONLY.DENY,
		},
		isProxy: Boolean,
	},
	emits: [
		"update:modbus",
		"update:id",
		"update:host",
		"update:port",
		"update:device",
		"update:baudrate",
		"update:comset",
		"update:readonly",
	],
	data() {
		return {
			connection: "tcpip" as ConnectionOption,
			protocol: "tcp" as ProtocolOption,
			MODBUS_PROXY_READONLY,
			MODBUS_BAUDRATE,
			MODBUS_COMSET,
		};
	},
	computed: {
		selectedModbus(): Modbus {
			if (this.connection === "serial") {
				return "rs485serial";
			}
			return this.protocol === "rtu" ? "rs485tcpip" : "tcpip";
		},
		showConnectionOptions() {
			return this.capabilities.includes("rs485");
		},
		showProtocolOptions() {
			return this.connection === "tcpip" && this.capabilities.includes("rs485");
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
		readonlyOptions() {
			return [
				{
					value: MODBUS_PROXY_READONLY.TRUE,
					name: this.$t("config.modbus.readonlyOptionSilent"),
				},
				{
					value: MODBUS_PROXY_READONLY.DENY,
					name: this.$t("config.modbus.readonlyOptionError"),
				},
				{
					value: MODBUS_PROXY_READONLY.FALSE,
					name: this.$t("config.modbus.readonlyOptionNo"),
				},
			];
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
	},
	mounted() {
		this.setConnectionAndProtocolByModbus(this.modbus);
		this.$emit("update:modbus", this.selectedModbus);
	},
	methods: {
		setProtocolByCapabilities(capabilities: ModbusCapability[]) {
			this.protocol = capabilities.includes("tcpip") ? "tcp" : "rtu";
		},
		setConnectionAndProtocolByModbus(modbus?: Modbus) {
			switch (modbus) {
				case "rs485serial":
					this.connection = "serial";
					this.protocol = "rtu";
					break;
				case "rs485tcpip":
					this.connection = "tcpip";
					this.protocol = "rtu";
					break;
				case "tcpip":
					this.connection = "tcpip";
					this.protocol = "tcp";
					break;
			}
		},
		formId(name: string): string {
			return `${name}-${this.id}`;
		},
	},
});
</script>
