<template>
	<JsonModal
		id="modbusProxyModal"
		:title="$t('config.modbusproxy.title')"
		:description="$t('config.modbusproxy.description')"
		docs="/docs/reference/configuration/modbusproxy"
		endpoint="/config/modbusproxy"
		state-key="modbusproxy"
		disable-remove
		data-testid="modbusproxy-modal"
		@changed="$emit('changed')"
	>
		<template #default="{ values }: { values: ModbusConnectionMap }">
			<div class="mb-3">
				<h6>Network Connection</h6>
				<p>
					The device is directly connected via a native network connection (Modbus TCP).
				</p>
				<div v-for="(connection, index) in networkConnections(values)">
					<div class="d-none d-lg-block">
						<hr class="mt-5" />
						<h5>
							<div class="inner mb-3">Network Connection #{{ index + 1 }}</div>
						</h5>
					</div>

					<FormRow id="serialConnectionURI" label="URI" example="192.0.2.2:502">
						<input
							id="serialConnectionURI"
							v-model="connection.URI"
							class="form-control"
							required
						/>
					</FormRow>
					<FormRow id="serialConnectionPort" label="Port" example="5022">
						<input
							id="serialConnectionPort"
							v-model="connection.Port"
							class="form-control"
							required
						/>
					</FormRow>
					<div class="align-items-center d-flex mb-4 justify-content-between">
						<div>
							<input
								id="serialConnectionRTU"
								v-model="connection.RTU"
								class="form-check-input"
								type="checkbox"
							/>
							<label class="form-check-label ms-2" for="serialConnectionRTU">
								Use Modbus RTU over TCP
							</label>
						</div>
						<button
							type="button"
							class="btn btn-sm btn-outline-secondary border-0"
							aria-label="Remove"
							tabindex="0"
							@click="removeConnection(values, connection)"
						>
							<shopicon-regular-trash
								size="s"
								class="flex-shrink-0"
							></shopicon-regular-trash>
						</button>
					</div>
				</div>
				<div class="d-flex align-items-center">
					<button
						type="button"
						class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
						data-testid="networkconnection-add"
						tabindex="0"
						@click="addConnection(values, DEFAULT_NETWORK_CONNECTION)"
					>
						<shopicon-regular-plus
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-plus>
						Add network connection
					</button>
				</div>
			</div>
			<div>
				<h6>Serial Connection</h6>
				<p>
					The device is connected directly via an RS485 adapter (Modbus RTU). Check the
					device configuration and read the relevant user manuals, data sheets, or system
					settings for further details.
				</p>
				<div v-for="(connection, index) in serialConnections(values)">
					<div class="d-none d-lg-block">
						<hr class="mt-5" />
						<h5>
							<div class="inner mb-3">Serial Connection #{{ index + 1 }}</div>
						</h5>
					</div>

					<FormRow id="serialConnectionDevice" label="Device" example="/dev/ttyUSB0">
						<input
							id="serialConnectionDevice"
							v-model="connection.Device"
							class="form-control"
							required
						/>
					</FormRow>
					<FormRow id="serialConnectionPort" label="Port" example="5022">
						<input
							id="serialConnectionPort"
							v-model="connection.Port"
							class="form-control"
							required
						/>
					</FormRow>
					<FormRow id="serialConnectionBaudrate" label="Baudrate" example="38400">
						<input
							id="serialConnectionBaudrate"
							v-model="connection.Baudrate"
							class="form-control"
							required
						/>
					</FormRow>
					<FormRow id="serialConnectionComset" label="Comset" example="8N1">
						<input
							id="serialConnectionComset"
							v-model="connection.Comset"
							class="form-control"
							required
						/>
					</FormRow>
					<div class="align-items-center d-flex mb-4">
						<button
							v-if="index + 1 === serialConnections(values).length"
							type="button"
							class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
							data-testid="networkconnection-add"
							tabindex="0"
							@click="addConnection(values, DEFAULT_SERIAL_CONNECTION)"
						>
							<shopicon-regular-plus
								size="s"
								class="flex-shrink-0"
							></shopicon-regular-plus>
							Add serial connection
						</button>
						<button
							type="button"
							class="btn btn-sm btn-outline-secondary border-0 ms-auto"
							aria-label="Remove"
							tabindex="0"
							@click="removeConnection(values, connection)"
						>
							<shopicon-regular-trash
								size="s"
								class="flex-shrink-0"
							></shopicon-regular-trash>
						</button>
					</div>
				</div>
			</div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/trash";
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";
import type {
	ModbusProxy,
	ModbusProxyNetworkConnection,
	ModbusProxySerialConnection,
} from "@/types/evcc";

type ModbusConnectionMap = Record<number, ModbusProxy>;

const DEFAULT_NETWORK_CONNECTION: ModbusProxyNetworkConnection = {
	Port: 0,
	RTU: false,
	URI: "",
};
const DEFAULT_SERIAL_CONNECTION: ModbusProxySerialConnection = {
	Baudrate: 0,
	Comset: "",
	Device: "",
	Port: 0,
};

export default {
	name: "ModbusProxyModal",
	components: { JsonModal, FormRow },
	emits: ["changed"],
	data() {
		return {
			DEFAULT_NETWORK_CONNECTION,
			DEFAULT_SERIAL_CONNECTION,
		};
	},
	computed: {
		networkConnections() {
			return (values: ModbusConnectionMap) => {
				return Object.values(values).filter((item) => "URI" in item);
			};
		},
		serialConnections() {
			return (values: ModbusConnectionMap) => {
				return Object.values(values).filter((item) => "Device" in item);
			};
		},
	},
	methods: {
		addConnection(values: ModbusConnectionMap, c: ModbusProxy) {
			values[Object.keys(values).length] = { ...c };
		},
		removeConnection(values: ModbusConnectionMap, c: ModbusProxy) {
			const entries = Object.values(values).filter((item) => item !== c);
			Object.keys(values).forEach((key) => delete values[Number(key)]);
			entries.forEach((item, i) => {
				values[i] = item;
			});
		},
	},
};
</script>

<style scoped>
h5 {
	position: relative;
	display: flex;
	top: -25px;
	margin-bottom: -0.5rem;
	padding: 0 0.5rem;
	justify-content: center;
}
h5 .inner {
	padding: 0 0.5rem;
	background-color: var(--evcc-box);
	font-weight: normal;
	color: var(--evcc-gray);
	text-transform: uppercase;
	text-align: center;
}
</style>
