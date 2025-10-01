<template>
	<JsonModal
		id="modbusProxyModal"
		:title="$t('config.modbusproxy.title')"
		:description="$t('config.modbusproxy.description')"
		docs="/docs/reference/configuration/modbusproxy"
		endpoint="/config/modbusproxy"
		state-key="modbusproxy"
		:store-values-in-array="true"
		disable-remove
		data-testid="modbusproxy-modal"
		@changed="$emit('changed')"
	>
		<template #default="{ values }: { values: ModbusProxy[] }">
			<div class="mb-3">
				<h6>Network Connection</h6>
				<p>
					The device is directly connected via a native network connection (Modbus TCP).
				</p>
				<div
					v-for="(connection, index) in values.filter(
						(item) => 'URI' in item.Settings
					) as ModbusProxy<ModbusProxyNetworkConnection>[]"
					:key="index"
				>
					<div class="d-none d-lg-block">
						<hr class="mt-5" />
						<h5>
							<div class="inner mb-3">Network Connection #{{ index + 1 }}</div>
						</h5>
					</div>

					<FormRow
						id="networkConnectionURI"
						label="URI"
						example="192.0.2.2:502"
						help="The IP address and the port of the target device in common URI Scheme."
					>
						<input
							id="networkConnectionURI"
							v-model="connection.Settings.URI"
							class="form-control"
							required
						/>
					</FormRow>
					<FormRow
						id="networkConnectionPort"
						label="Port"
						example="5022"
						help="The local TCP/IP port under which a connection is provided as a proxy server."
					>
						<input
							id="networkConnectionPort"
							v-model="connection.Port"
							class="form-control"
							required
						/>
					</FormRow>
					<FormRow
						id="networkConnectionReadonly"
						label="Readonly"
						:help="
							connection.ReadOnly === MODBUS_PROXY_READONLY.TRUE
								? 'Write access is blocked without response.'
								: connection.ReadOnly === MODBUS_PROXY_READONLY.FALSE
									? 'Write access is blocked with a modbus error as response.'
									: connection.ReadOnly === MODBUS_PROXY_READONLY.DENY
										? 'Write access is forwarded.'
										: 'Whether Modbus write accesses by third-party systems should be blocked.'
						"
					>
						<SelectGroup
							id="networkConnectionReadonly"
							v-model="connection.ReadOnly"
							class="w-100"
							:options="
								Object.values(MODBUS_PROXY_READONLY).map((v) => ({
									value: v,
									name: v,
								}))
							"
							transparent
						/>
					</FormRow>
					<div class="align-items-center d-flex mb-4 justify-content-between">
						<div>
							<input
								id="networkConnectionRTU"
								v-model="connection.Settings.RTU"
								class="form-check-input"
								type="checkbox"
							/>
							<label class="form-check-label ms-2" for="networkConnectionRTU">
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
				<div
					v-for="(connection, index) in values.filter(
						(item) => 'Device' in item.Settings
					) as ModbusProxy<ModbusProxySerialConnection>[]"
					:key="index"
				>
					<div class="d-none d-lg-block">
						<hr class="mt-5" />
						<h5>
							<div class="inner mb-3">Serial Connection #{{ index + 1 }}</div>
						</h5>
					</div>

					<FormRow id="serialConnectionDevice" label="Device" example="/dev/ttyUSB0">
						<input
							id="serialConnectionDevice"
							v-model="connection.Settings.Device"
							class="form-control"
							required
						/>
					</FormRow>
					<FormRow
						id="serialConnectionPort"
						label="Port"
						example="5022"
						help="The local TCP/IP port under which a connection is provided as a proxy server."
					>
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
							v-model="connection.Settings.Baudrate"
							class="form-control"
							required
						/>
					</FormRow>
					<FormRow id="serialConnectionComset" label="Comset" example="8N1">
						<input
							id="serialConnectionComset"
							v-model="connection.Settings.Comset"
							class="form-control"
							required
						/>
					</FormRow>
					<FormRow
						id="serialConnectionReadonly"
						label="Readonly"
						:help="
							connection.ReadOnly === MODBUS_PROXY_READONLY.TRUE
								? 'Write access is blocked without response.'
								: connection.ReadOnly === MODBUS_PROXY_READONLY.FALSE
									? 'Write access is blocked with a modbus error as response.'
									: connection.ReadOnly === MODBUS_PROXY_READONLY.DENY
										? 'Write access is forwarded.'
										: 'Whether Modbus write accesses by third-party systems should be blocked.'
						"
					>
						<SelectGroup
							id="serialConnectionReadonly"
							v-model="connection.ReadOnly"
							class="w-100"
							:options="
								Object.values(MODBUS_PROXY_READONLY).map((v) => ({
									value: v,
									name: v,
								}))
							"
							transparent
						/>
					</FormRow>
					<div class="align-items-center d-flex mb-4">
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
				<button
					type="button"
					class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
					data-testid="networkconnection-add"
					tabindex="0"
					@click="addConnection(values, DEFAULT_SERIAL_CONNECTION)"
				>
					<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
					Add serial connection
				</button>
			</div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/trash";
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";
import {
	MODBUS_PROXY_READONLY,
	type ModbusProxy,
	type ModbusProxyNetworkConnection,
	type ModbusProxySerialConnection,
} from "@/types/evcc";
import SelectGroup from "../Helper/SelectGroup.vue";

const DEFAULT_NETWORK_CONNECTION: ModbusProxy<ModbusProxyNetworkConnection> = {
	Port: 0,
	ReadOnly: MODBUS_PROXY_READONLY.TRUE,
	Settings: { RTU: false, URI: "" },
};
const DEFAULT_SERIAL_CONNECTION: ModbusProxy<ModbusProxySerialConnection> = {
	Port: 0,
	ReadOnly: MODBUS_PROXY_READONLY.TRUE,
	Settings: { Baudrate: 0, Comset: "", Device: "" },
};

export default {
	name: "ModbusProxyModal",
	components: { JsonModal, FormRow, SelectGroup },
	emits: ["changed"],
	data() {
		return {
			DEFAULT_NETWORK_CONNECTION,
			DEFAULT_SERIAL_CONNECTION,
			MODBUS_PROXY_READONLY,
		};
	},
	methods: {
		addConnection(values: ModbusProxy[], c: ModbusProxy) {
			values[Object.keys(values).length] = { ...c };
		},
		removeConnection(values: ModbusProxy[], c: ModbusProxy) {
			values.splice(values.indexOf(c), 1);
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
