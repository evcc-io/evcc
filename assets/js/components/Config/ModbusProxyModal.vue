<template>
	<JsonModal
		id="modbusProxyModal"
		:title="$t('config.modbusproxy.title')"
		:description="$t('config.modbusproxy.description')"
		docs="/docs/reference/configuration/modbusproxy"
		endpoint="/config/modbusproxy"
		state-key="modbusproxy"
		store-values-in-array
		disable-remove
		data-testid="modbusproxy-modal"
		size="xl"
		@changed="$emit('changed')"
	>
		<template #default="{ values }: { values: ModbusProxy[] }">
			<div class="mb-3">
				<SponsorTokenRequired v-if="!isSponsor" feature />
				<pre class="text-monospace my-5">{{ ASCII_DIAGRAM }}</pre>
				<div v-for="(c, index) in values" :key="index" data-testid="modbusproxy-connection">
					<div class="d-block">
						<hr class="mt-5" />
						<h5>
							<div class="inner mb-4">
								{{ $t("config.modbusproxy.connection", { number: index + 1 }) }}
							</div>
						</h5>
					</div>
					<div class="row d-inline d-lg-flex mb-3">
						<div class="col-lg-5" data-testid="evcc-box">
							<div class="border rounded px-3 pt-4 pb-3">
								<div class="d-lg-block">
									<h5 class="box-heading">
										<div class="inner">evcc</div>
									</h5>
								</div>
								<FormRow
									:id="formId(index, 'sourcePort')"
									:label="$t('config.modbus.port')"
									:help="$t('config.modbusproxy.sourcePortHelp')"
								>
									<PropertyField
										:id="formId(index, 'sourcePort')"
										v-model="c.port"
										property="port"
										type="Int"
										class="w-50"
										required
									/>
								</FormRow>
								<FormRow
									:id="formId(index, 'readonly')"
									:label="$t('config.modbusproxy.readonly.label')"
									:help="getReadonlyHelp(c.readonly)"
								>
									<SelectGroup
										:id="formId(index, 'readonly')"
										v-model="c.readonly"
										class="w-100"
										:options="readonlyOptions"
										transparent
									/>
								</FormRow>
							</div>
						</div>
						<div
							class="col-lg-2 d-none d-lg-flex justify-content-center evcc-gray"
							style="padding-top: 2.5rem"
						>
							<shopicon-regular-arrowright
								size="l"
								class="flex-shrink-0"
							></shopicon-regular-arrowright>
						</div>
						<div class="col d-flex d-lg-none justify-content-center evcc-gray my-3">
							<shopicon-regular-arrowdown
								size="l"
								class="flex-shrink-0"
							></shopicon-regular-arrowdown>
						</div>
						<div class="col-lg-5" data-testid="device-box">
							<div class="border rounded px-3 pt-4 pb-3">
								<div class="d-lg-block">
									<h5 class="box-heading">
										<div class="inner">
											{{ $t("config.modbusproxy.device") }}
										</div>
									</h5>
								</div>
								<Modbus
									v-model:baudrate="c.settings.baudrate"
									v-model:comset="c.settings.comset"
									v-model:device="c.settings.device"
									:modbus="getModbus(c.settings)"
									:component-id="`proxy-${index}`"
									:host="getHost(c.settings.uri)"
									:port="getPort(c.settings.uri)"
									:capabilities="['rs485', 'tcpip']"
									hide-modbus-id
									@update:host="(host) => updateHost(host, c.settings)"
									@update:port="(port) => updatePort(port, c.settings)"
									@update:modbus="(modbus) => updateModbus(c.settings, modbus)"
								/>
							</div>
						</div>
					</div>
					<button
						type="button"
						class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray ms-auto"
						:aria-label="$t('config.general.remove')"
						tabindex="0"
						@click="values.splice(index, 1)"
					>
						<shopicon-regular-trash
							size="s"
							class="flex-shrink-0"
						></shopicon-regular-trash>
						{{ $t("config.general.remove") }}
					</button>
				</div>
				<hr class="my-5" />
				<button
					type="button"
					class="d-flex btn btn-sm align-items-center gap-2 mb-5"
					:class="
						values.length === 0
							? 'btn-secondary'
							: 'btn-outline-secondary border-0 evcc-gray'
					"
					data-testid="networkconnection-add"
					tabindex="0"
					@click="addConnection(values)"
				>
					<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
					{{ $t("config.modbusproxy.add") }}
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
import {
	MODBUS_COMSET,
	MODBUS_CONNECTION,
	MODBUS_PROTOCOL,
	MODBUS_PROXY_READONLY,
	MODBUS_TYPE,
	type ModbusProxy,
	type ModbusProxySettings,
} from "@/types/evcc";
import ASCII_DIAGRAM from "./modbus-diagram.txt?raw";
import Modbus from "./DeviceModal/Modbus.vue";
import PropertyField from "./PropertyField.vue";
import FormRow from "./FormRow.vue";
import SponsorTokenRequired from "./DeviceModal/SponsorTokenRequired.vue";
import SelectGroup from "@/components/Helper/SelectGroup.vue";
import { defineComponent } from "vue";

export default defineComponent({
	name: "ModbusProxyModal",
	components: { JsonModal, Modbus, FormRow, PropertyField, SponsorTokenRequired, SelectGroup },
	props: {
		isSponsor: Boolean,
	},
	emits: ["changed"],
	data() {
		return {
			ASCII_DIAGRAM,
			MODBUS_PROXY_READONLY,
			MODBUS_CONNECTION,
			MODBUS_PROTOCOL,
			MODBUS_TYPE,
		};
	},
	computed: {
		readonlyOptions() {
			return Object.values(MODBUS_PROXY_READONLY).map((value) => ({
				value,
				name: this.$t(`config.modbusproxy.option.${value}`),
			}));
		},
	},
	methods: {
		formId(index: number, name: string) {
			return `modbusproxy-connection-${index}-${name}`;
		},
		getModbus(s: ModbusProxySettings) {
			if (s.device) {
				return MODBUS_TYPE.RS485_SERIAL;
			}
			return s.rtu ? MODBUS_TYPE.RS485_TCPIP : MODBUS_TYPE.TCPIP;
		},
		getReadonlyHelp(readonly = MODBUS_PROXY_READONLY.FALSE): string {
			return this.$t(`config.modbusproxy.readonly.help.${readonly}`);
		},
		addConnection(values: ModbusProxy[]) {
			const highestPort = values.length > 0 ? Math.max(...values.map((c) => c.port)) : 1501;
			values.push({
				port: highestPort + 1,
				readonly: MODBUS_PROXY_READONLY.FALSE,
				settings: {
					uri: ":502",
					baudrate: 9600,
					comset: "8N1" as MODBUS_COMSET,
				},
			});
		},
		getHost(uri?: string) {
			return uri?.split(":")[0] || "";
		},
		getPort(uri?: string) {
			return uri?.split(":")[1] || "";
		},
		updateHost(newHost: string, settings: ModbusProxySettings) {
			const port = this.getPort(settings.uri);
			settings.uri = `${newHost}:${port}`;
		},
		updatePort(newPort: string | number, settings: ModbusProxySettings) {
			const host = this.getHost(settings.uri);
			settings.uri = `${host}:${newPort}`;
		},
		updateModbus(settings: ModbusProxySettings, modbus: MODBUS_TYPE) {
			switch (modbus) {
				case MODBUS_TYPE.RS485_SERIAL:
					settings.uri = undefined;
					settings.rtu = undefined;
					break;
				case MODBUS_TYPE.RS485_TCPIP:
				case MODBUS_TYPE.TCPIP:
					settings.device = undefined;
					settings.baudrate = undefined;
					settings.comset = undefined;
					settings.rtu = modbus === MODBUS_TYPE.RS485_TCPIP;
					break;
			}
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
h5.box-heading {
	top: -34px;
	margin-bottom: -24px;
}
h5 .inner {
	padding: 0 0.5rem;
	background-color: var(--evcc-box);
	font-weight: normal;
	color: var(--evcc-gray);
	text-align: center;
}
</style>
