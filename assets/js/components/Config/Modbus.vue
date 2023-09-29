<template>
	<FormRow v-if="showConnectionOptions" id="modbusTcp" label="Modbus Interface">
		<div class="btn-group" role="group">
			<input
				id="modbusTcp"
				v-model="connection"
				type="radio"
				class="btn-check"
				name="modbusConnection"
				value="tcp"
				autocomplete="off"
			/>
			<label class="btn btn-outline-primary" for="modbusTcp">Network</label>
			<input
				id="modbusSerial"
				v-model="connection"
				type="radio"
				class="btn-check"
				name="modbusConnection"
				value="serial"
				autocomplete="off"
			/>
			<label class="btn btn-outline-primary" for="modbusSerial">Serial / USB</label>
		</div>
	</FormRow>
	<FormRow id="modbusId" label="Modbus ID">
		<PropertyField
			id="modbusId"
			property="id"
			type="Number"
			class="me-2"
			required
			@change="$emit('update:id', $event.target.value)"
		/>
	</FormRow>
	<div v-if="connection === 'tcp'">
		<FormRow id="modbusHost" label="Modbus Host/IP">
			<PropertyField
				id="modbusHost"
				property="host"
				type="String"
				class="me-2"
				required
				@change="$emit('update:host', $event.target.value)"
			/>
		</FormRow>
		<FormRow id="modbusPort" label="Modbus Port">
			<PropertyField
				id="modbusPort"
				property="port"
				type="Number"
				class="me-2"
				required
				@change="$emit('update:port', $event.target.value)"
			/>
		</FormRow>
		<FormRow
			v-if="showProtocolOptions"
			id="modbusAscii"
			label="Modbus Protocol"
			:help="
				protocol === 'ascii'
					? 'Direkte Netzwerkverbindung über LAN/Wifi'
					: 'Über RS485 zu Ethernet Adapter'
			"
		>
			<div class="btn-group" role="group">
				<input
					id="modbusAscii"
					v-model="protocol"
					type="radio"
					class="btn-check"
					name="modbusAscii"
					value="ascii"
					autocomplete="off"
				/>
				<label class="btn btn-outline-primary" for="modbusAscii">ASCII</label>
				<input
					id="modbusRtu"
					v-model="protocol"
					type="radio"
					class="btn-check"
					name="modbusProtocol"
					value="rtu"
					autocomplete="off"
				/>
				<label class="btn btn-outline-primary" for="modbusRtu">RTU</label>
			</div>
		</FormRow>
	</div>
	<div v-else>
		<FormRow id="modbusDevice" label="Modbus Device">
			<PropertyField
				id="modbusDevice"
				property="device"
				type="String"
				class="me-2"
				required
				@change="$emit('update:device', $event.target.value)"
			/>
		</FormRow>
		<FormRow id="modbusBaudrate" label="Modbus Baudrate">
			<PropertyField
				id="modbusBaudrate"
				property="baudrate"
				type="Number"
				class="me-2"
				:valid-values="[9600, 19200, 38400, 57600, 115200]"
				required
				@change="$emit('update:baudrate', $event.target.value)"
			/>
		</FormRow>
		<FormRow id="modbusComset" label="Modbus Port">
			<PropertyField
				id="modbusComset"
				property="port"
				type="String"
				class="me-2"
				:valid-values="['8N1', '8E1']"
				required
				@change="$emit('update:comset', $event.target.value)"
			/>
		</FormRow>
	</div>
</template>

<script>
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";

export default {
	name: "Modbus",
	components: { FormRow, PropertyField },
	props: {
		options: Array,
		modbus: String,
		id: Number,
		host: String,
		port: Number,
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
			connection: "tcp", // serial, tcp
			protocol: "ascii", // rtu, ascii
		};
	},
	computed: {
		selectedModbus() {
			if (this.connection === "serial") {
				return "rs485serial";
			}
			return this.protocol === "rtu" ? "rs485tcpip" : "tcpip";
		},
		showConnectionOptions() {
			return this.options.includes("rs485");
		},
		showProtocolOptions() {
			return this.connection === "tcp" && this.options.includes("tcpip");
		},
	},
	watch: {
		selectedModbus(newValue) {
			this.$emit("update:modbus", newValue);
		},
		options(newValue) {
			this.setConnectionAndProtocolByModbus(newValue[0]);
			this.$emit("update:modbus", this.selectedModbus);
		},
		modbus(newValue) {
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
		setConnectionAndProtocolByModbus(modbus) {
			console.log("setConnectionAndProtocolByModbus", modbus);
			switch (modbus) {
				case "rs485serial":
					this.connection = "serial";
					this.protocol = "rtu";
					break;
				case "rs485tcpip":
					this.connection = "tcp";
					this.protocol = "rtu";
					break;
				case "tcpip":
					this.connection = "tcp";
					this.protocol = "ascii";
					break;
				default:
					this.connection = "tcp";
					this.protocol = "ascii";
			}
		},
	},
};
</script>
