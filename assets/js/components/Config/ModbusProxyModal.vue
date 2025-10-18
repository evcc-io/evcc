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
		size="lg"
		@changed="$emit('changed')"
	>
		<template #default="{ values }: { values: ModbusProxy[] }">
			<div class="mb-3">
				<pre class="text-monospace">{{ ASCII_DIAGRAM }}</pre>
				<div v-for="(connection, index) in values" :key="index">
					<div class="d-block">
						<hr class="mt-5" />
						<h5>
							<div class="inner mb-3">Modbus Proxy Connection #{{ index + 1 }}</div>
						</h5>
					</div>
					<div class="row d-inline d-lg-flex mb-3">
						<div class="col-lg-4">
							<div class="border rounded px-3">
								<div class="d-lg-block">
									<hr class="mt-4" />
									<h5>
										<div class="inner">Client</div>
									</h5>
								</div>
								<FormRow id="modbusPort" :label="$t('config.modbus.port')">
									<PropertyField
										id="modbusPort"
										v-model="connection.Port"
										property="port"
										type="Int"
										class="w-50"
										required
									/>
								</FormRow>
							</div>
						</div>
						<div
							class="col d-none d-lg-flex justify-content-center evcc-gray"
							style="padding-top: 9.5%"
						>
							<shopicon-regular-arrowright
								size="l"
								class="flex-shrink-0"
							></shopicon-regular-arrowright>
						</div>
						<div class="col d-flex d-lg-none justify-content-center evcc-gray">
							<shopicon-regular-arrowdown
								size="l"
								class="flex-shrink-0"
							></shopicon-regular-arrowdown>
						</div>
						<div class="col-lg-6">
							<div class="border rounded px-3">
								<div class="d-lg-block">
									<hr class="mt-4" />
									<h5>
										<div class="inner">Device</div>
									</h5>
								</div>
								<Modbus
									:id="index"
									v-model:baudrate="connection.Settings.Baudrate"
									v-model:comset="connection.Settings.Comset"
									v-model:device="connection.Settings.Device"
									v-model:readonly="connection.ReadOnly"
									:host="getHost(connection.Settings.URI)"
									:port="getPort(connection.Settings.URI)"
									:capabilities="['rs485', 'tcpip']"
									:is-proxy="true"
									@update:host="updateHost(connection, $event)"
									@update:port="updatePort(connection, $event)"
								/>
							</div>
						</div>
					</div>
					<button
						type="button"
						class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray ms-auto"
						aria-label="Remove"
						tabindex="0"
						@click="values.splice(index, 1)"
					>
						<shopicon-regular-trash
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-trash>
						Remove connection
					</button>
				</div>
				<hr />
				<button
					type="button"
					class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
					data-testid="networkconnection-add"
					tabindex="0"
					@click="values.push(DEFAULT_MODBUS_PROXY)"
				>
					<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
					Add network connection
				</button>
			</div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/arrowright";
import "@h2d2/shopicons/es/regular/arrowdown";
import "@h2d2/shopicons/es/regular/plus";
import "@h2d2/shopicons/es/regular/trash";
import JsonModal from "./JsonModal.vue";
import { MODBUS_PROXY_READONLY, type ModbusProxy } from "@/types/evcc";
import ASCII_DIAGRAM from "./modbus-diagram.txt?raw";
import Modbus from "./DeviceModal/Modbus.vue";
import PropertyField from "./PropertyField.vue";
import FormRow from "./FormRow.vue";
import { defineComponent } from "vue";

const DEFAULT_MODBUS_PROXY: ModbusProxy = {
	Port: 502,
	ReadOnly: MODBUS_PROXY_READONLY.DENY,
	Settings: {},
};

export default defineComponent({
	name: "ModbusProxyModal",
	components: { JsonModal, Modbus, FormRow, PropertyField },
	emits: ["changed"],
	data() {
		return {
			ASCII_DIAGRAM,
			MODBUS_PROXY_READONLY,
			DEFAULT_MODBUS_PROXY,
		};
	},
	methods: {
		getHost(uri?: string) {
			return uri?.split(":")[0] || "";
		},
		getPort(uri?: string) {
			return uri?.split(":")[1] || "";
		},
		updateHost(connection: ModbusProxy, newHost: string) {
			const port = this.getPort(connection.Settings.URI);
			connection.Settings.URI = `${newHost}:${port}`;
		},
		updatePort(connection: ModbusProxy, newPort: string | number) {
			const host = this.getHost(connection.Settings.URI);
			connection.Settings.URI = `${host}:${newPort}`;
		},
	},
});
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
