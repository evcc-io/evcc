<template>
	<GenericModal
		id="chargerModal"
		:title="modalTitle"
		data-testid="charger-modal"
		:fade="fade"
		@open="open"
		@close="close"
	>
		<form ref="form" class="container mx-0 px-0">
			<FormRow id="chargerTemplate" :label="$t('config.charger.template')">
				<select
					id="chargerTemplate"
					v-model="templateName"
					:disabled="!isNew"
					class="form-select w-100"
					@change="templateChanged"
				>
					<option value="">---</option>
					<optgroup :label="$t('config.charger.generic')">
						<option
							v-for="option in genericOptions"
							:key="option.name"
							:value="option.template"
						>
							{{ option.name }}
						</option>
					</optgroup>
					<optgroup :label="$t('config.charger.chargers')">
						<option
							v-for="option in chargerOptions"
							:key="option.name"
							:value="option.template"
						>
							{{ option.name }}
						</option>
					</optgroup>
					<optgroup :label="$t('config.charger.switchsocket')">
						<option
							v-for="option in switchSocketOptions"
							:key="option.name"
							:value="option.template"
						>
							{{ option.name }}
						</option>
					</optgroup>
				</select>
			</FormRow>
			<p v-if="loadingTemplate">Loading ...</p>
			<SponsorTokenRequired v-if="sponsorTokenRequired" />
			<Markdown v-if="description" :markdown="description" class="my-4" />
			<FormRow
				v-if="ocppUrl"
				id="chargerOcppUrl"
				:label="$t('config.charger.ocppLabel')"
				:help="$t('config.charger.ocppHelp')"
			>
				<input
					type="text"
					class="form-control border border-success bg-transparent"
					:value="ocppUrl"
					readonly
				/>
			</FormRow>

			<Modbus
				v-if="modbus"
				v-model:modbus="values.modbus"
				v-model:id="values.id"
				v-model:host="values.host"
				v-model:port="values.port"
				v-model:device="values.device"
				v-model:baudrate="values.baudrate"
				v-model:comset="values.comset"
				:defaultId="modbus.ID"
				:defaultComset="modbus.Comset"
				:defaultBaudrate="modbus.Baudrate"
				:defaultPort="modbus.Port"
				:capabilities="modbusCapabilities"
			/>
			<PropertyEntry
				v-for="param in normalParams"
				:id="`chargerParam${param.Name}`"
				:key="param.Name"
				v-bind="param"
				v-model="values[param.Name]"
			/>

			<PropertyCollapsible>
				<template v-if="advancedParams.length" #advanced>
					<PropertyEntry
						v-for="param in advancedParams"
						:id="`chargerParam${param.Name}`"
						:key="param.Name"
						v-bind="param"
						v-model="values[param.Name]"
					/>
				</template>
			</PropertyCollapsible>

			<TestResult
				v-if="templateName"
				:success="testSuccess"
				:failed="testFailed"
				:unknown="testUnknown"
				:running="testRunning"
				:result="testResult"
				:error="testError"
				@test="testManually"
			/>

			<div v-if="templateName" class="my-4 d-flex justify-content-between">
				<button
					v-if="isDeletable"
					type="button"
					class="btn btn-link text-danger"
					@click.prevent="remove"
				>
					{{ $t("config.general.delete") }}
				</button>
				<button
					v-else
					type="button"
					class="btn btn-link text-muted"
					data-bs-dismiss="modal"
					tabindex="0"
				>
					{{ $t("config.general.cancel") }}
				</button>
				<button
					type="submit"
					class="btn btn-primary"
					:disabled="testRunning || saving || sponsorTokenRequired"
					tabindex="0"
					@click.prevent="isNew ? create() : update()"
				>
					<span
						v-if="saving"
						class="spinner-border spinner-border-sm"
						role="status"
						aria-hidden="true"
					></span>
					{{
						testUnknown ? $t("config.general.validateSave") : $t("config.general.save")
					}}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script>
import FormRow from "./FormRow.vue";
import PropertyEntry from "./PropertyEntry.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import TestResult from "./TestResult.vue";
import api from "../../api";
import test from "./mixins/test";
import Modbus from "./Modbus.vue";
import GenericModal from "../GenericModal.vue";
import Markdown from "./Markdown.vue";
import SponsorTokenRequired from "./SponsorTokenRequired.vue";
const initialValues = { type: "template" };

function sleep(ms) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

const CUSTOM_FIELDS = ["modbus"];

export default {
	name: "ChargerModal",
	components: {
		FormRow,
		PropertyEntry,
		GenericModal,
		Modbus,
		TestResult,
		PropertyCollapsible,
		Markdown,
		SponsorTokenRequired,
	},
	mixins: [test],
	props: {
		id: Number,
		name: String,
		fade: String,
		isSponsor: Boolean,
	},
	emits: ["added", "updated", "removed", "close"],
	data() {
		return {
			isModalVisible: false,
			templates: [],
			products: [],
			templateName: null,
			template: null,
			saving: false,
			selectedType: null,
			loadingTemplate: false,
			values: { ...initialValues },
		};
	},
	computed: {
		modalTitle() {
			if (this.isNew) {
				return this.$t(`config.charger.titleAdd`);
			}
			return this.$t(`config.charger.titleEdit`);
		},
		chargerOptions() {
			return this.products.filter((p) => !p.group);
		},
		genericOptions() {
			return this.products.filter((p) => p.group === "generic");
		},
		switchSocketOptions() {
			return this.products.filter((p) => p.group === "switchsockets");
		},
		templateParams() {
			const params = this.template?.Params || [];
			return params.filter((p) => !CUSTOM_FIELDS.includes(p.Name));
		},
		normalParams() {
			return this.templateParams.filter((p) => !p.Advanced);
		},
		advancedParams() {
			return this.templateParams.filter((p) => p.Advanced);
		},
		modbus() {
			const params = this.template?.Params || [];
			return params.find((p) => p.Name === "modbus");
		},
		modbusCapabilities() {
			return this.modbus?.Choice || [];
		},
		ocppUrl() {
			const isOcpp =
				this.templateParams.some((p) => p.Name === "connector") &&
				this.templateParams.some((p) => p.Name === "stationid");
			if (isOcpp) {
				return `ws://${window.location.hostname}:8887`;
			}
			return null;
		},
		modbusDefaults() {
			const { ID, Comset, Baudrate, Port } = this.modbus || {};
			return {
				id: ID,
				comset: Comset,
				baudrate: Baudrate,
				port: Port,
			};
		},
		description() {
			return this.template?.Requirements?.Description;
		},
		sponsorTokenRequired() {
			const list = this.template?.Requirements?.EVCC || [];
			return list.includes("sponsorship") && !this.isSponsor;
		},
		apiData() {
			return {
				template: this.templateName,
				...this.modbusDefaults,
				...this.values,
			};
		},
		isNew() {
			return this.id === undefined;
		},
		isDeletable() {
			return !this.isNew;
		},
	},
	watch: {
		isModalVisible(visible) {
			if (visible) {
				this.templateName = null;
				this.reset();
				this.loadProducts();
				if (this.id !== undefined) {
					this.loadConfiguration();
				}
			}
		},
		templateName() {
			this.loadTemplate();
		},
		values: {
			handler() {
				this.resetTest();
			},
			deep: true,
		},
	},
	methods: {
		reset() {
			this.values = { ...initialValues };
			this.resetTest();
		},
		async loadConfiguration() {
			try {
				const charger = (await api.get(`config/devices/charger/${this.id}`)).data.result;
				this.values = charger.config;
				this.applyDefaultsFromTemplate();
				this.templateName = this.values.template;
			} catch (e) {
				console.error(e);
			}
		},
		async loadProducts() {
			if (!this.isModalVisible) {
				return;
			}
			try {
				const opts = { params: { lang: this.$i18n?.locale } };
				this.products = (await api.get("config/products/charger", opts)).data.result;
			} catch (e) {
				console.error(e);
			}
		},
		async loadTemplate() {
			this.template = null;
			this.loadingTemplate = true;
			try {
				const opts = {
					params: {
						lang: this.$i18n?.locale,
						name: this.templateName,
					},
				};
				const result = await api.get("config/templates/charger", opts);
				this.template = result.data.result;
				this.applyDefaultsFromTemplate();
			} catch (e) {
				console.error(e);
			}
			this.loadingTemplate = false;
		},
		applyDefaultsFromTemplate() {
			const params = this.template?.Params || [];
			params
				.filter((p) => p.Default && !this.values[p.Name])
				.forEach((p) => {
					this.values[p.Name] = p.Default;
				});
		},
		async create() {
			if (this.testUnknown) {
				const success = await this.test(this.testCharger);
				if (!success) return;
				await sleep(100);
			}
			this.saving = true;
			try {
				const response = await api.post("config/devices/charger", this.apiData);
				const { name } = response.data.result;
				this.$emit("added", name);
				this.$emit("updated");
				this.close();
			} catch (e) {
				console.error(e);
				alert("create failed");
			}
			this.saving = false;
		},
		async testManually() {
			await this.test(this.testCharger);
		},
		async testCharger() {
			let url = "config/test/charger";
			if (!this.isNew) {
				url += `/merge/${this.id}`;
			}
			return await api.post(url, this.apiData);
		},
		async update() {
			if (this.testUnknown) {
				const success = await this.test(this.testCharger);
				if (!success) return;
				await sleep(250);
			}
			this.saving = true;
			try {
				await api.put(`config/devices/charger/${this.id}`, this.apiData);
				this.$emit("updated");
				this.close();
			} catch (e) {
				console.error(e);
				alert("update failed");
			}
			this.saving = false;
		},
		async remove() {
			try {
				await api.delete(`config/devices/charger/${this.id}`);
				this.$emit("removed", this.name);
				this.$emit("updated");
				this.close();
			} catch (e) {
				console.error(e);
				alert("delete failed");
			}
		},
		open() {
			this.isModalVisible = true;
		},
		close() {
			this.$emit("close");
			this.isModalVisible = false;
		},
		templateChanged() {
			this.reset();
		},
	},
};
</script>
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
	padding-right: 0;
}
.addButton {
	min-height: auto;
}
</style>
