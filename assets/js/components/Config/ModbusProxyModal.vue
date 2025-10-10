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
				<pre class="text-monospace">{{ ASCII_DIAGRAM }}</pre>
				<div v-for="(connection, index) in values">
					<div class="d-none d-lg-block">
						<hr class="mt-5" />
						<h5>
							<div class="inner mb-3">Connection #{{ index + 1 }}</div>
						</h5>
					</div>
					<Modbus
						:capabilities="['rs485', 'tcpip']"
						:host="connection.Settings.URI"
						:port="connection.Port"
						:baudrate="connection.Settings.Baudrate"
						:comset="connection.Settings.Comset"
						:device="connection.Settings.Device"
						:read-only="connection.ReadOnly"
						:is-proxy="true"
					/>
					<div class="align-items-center d-flex mb-4">
						<button
							type="button"
							class="btn btn-sm btn-outline-secondary border-0 ms-auto"
							aria-label="Remove"
							tabindex="0"
							@click="values.splice(index, 1)"
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
					@click="addConnection(values)"
				>
					<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
					Add network connection
				</button>
			</div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/plus";
import "@h2d2/shopicons/es/regular/trash";
import JsonModal from "./JsonModal.vue";
import { MODBUS_PROXY_READONLY, type ModbusProxy } from "@/types/evcc";
import ASCII_DIAGRAM from "./modbus-diagram.txt?raw";
import Modbus from "./DeviceModal/Modbus.vue";
import deepClone from "@/utils/deepClone";

const DEFAULT_MODBUS_PROXY: ModbusProxy = {
	Port: 502,
	ReadOnly: MODBUS_PROXY_READONLY.DENY,
	Settings: {},
};

export default {
	name: "ModbusProxyModal",
	components: { JsonModal, Modbus },
	emits: ["changed"],
	data() {
		return {
			ASCII_DIAGRAM,
			MODBUS_PROXY_READONLY,
			DEFAULT_MODBUS_PROXY,
			deepClone,
		};
	},
	methods: {
		addConnection(values: ModbusProxy[]) {
			const newConnection = { ...deepClone(DEFAULT_MODBUS_PROXY) }; // Ensures reactivity with spread
			values.push(newConnection);
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
