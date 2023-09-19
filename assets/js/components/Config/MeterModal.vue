<template>
	<Teleport to="body">
		<div
			id="meterModal"
			ref="modal"
			class="modal fade text-dark"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							{{ modalTitle }}
						</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<form ref="form" class="container mx-0 px-0">
							<FormRow id="meterTemplate" :label="$t('config.meter.template')">
								<select
									id="meterTemplate"
									v-model="templateName"
									:disabled="!isNew"
									class="form-select w-100"
									@change="templateChanged"
								>
									<option
										v-for="option in templateOptions"
										:key="option.name"
										:value="option.template"
									>
										{{ option.name }}
									</option>
								</select>
							</FormRow>
							<FormRow
								v-for="param in templateParams"
								:id="`meterParam${param.Name}`"
								:key="param.Name"
								:optional="!param.Required"
								:label="param.Description || `[${param.Name}]`"
								:help="param.Description === param.Help ? undefined : param.Help"
								:small-value="['capacity'].includes(param.Name)"
								:example="param.Example"
							>
								<PropertyField
									:id="`meterParam${param.Name}`"
									v-model="values[param.Name]"
									:masked="param.Mask"
									:property="param.Name"
									class="me-2"
									:required="param.Required"
									:validValues="param.ValidValues"
								/>
							</FormRow>

							<TestResult
								v-if="testResult"
								:success="testSuccess"
								:failed="testFailed"
								:unknown="testUnknown"
								:result="testResult"
								@test="testManually"
							/>

							<div v-if="templateName" class="my-4 d-flex justify-content-between">
								<button
									v-if="isDeletable"
									type="button"
									class="btn btn-link text-danger"
									@click.prevent="remove"
								>
									{{ $t("config.meter.delete") }}
								</button>
								<button
									v-else
									type="button"
									class="btn btn-link text-muted"
									data-bs-dismiss="modal"
								>
									{{ $t("config.meter.cancel") }}
								</button>
								<button
									type="submit"
									class="btn btn-primary"
									:disabled="testRunning"
									@click.prevent="isNew ? create() : update()"
								>
									{{
										testUnknown
											? $t("config.meter.validateSave")
											: $t("config.meter.save")
									}}
								</button>
							</div>
						</form>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import TestResult from "./TestResult.vue";
import api from "../../api";
import test from "./mixins/test";

const initialValues = { type: "template", icon: "car" };

function sleep(ms) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

export default {
	name: "MeterModal",
	components: { FormRow, PropertyField, TestResult },
	mixins: [test],
	props: {
		id: Number,
		type: String,
	},
	emits: ["meter-changed"],
	data() {
		return {
			isModalVisible: false,
			templates: [],
			products: [],
			templateName: null,
			template: null,
			values: { ...initialValues },
		};
	},
	computed: {
		modalTitle() {
			if (this.isNew) {
				if (this.type) {
					return this.$t(`config.${this.type}.titleAdd`);
				} else {
					return this.$t("config.pvOrBattery.titleAdd");
				}
			}
			return this.$t(`config.${this.type}.titleEdit`);
		},
		templateOptions() {
			return this.products;
		},
		templateParams() {
			const params = this.template?.Params || [];
			const filteredParams = params.filter((p) => !p.Advanced || p.Name === "icon");
			const adjustedParams = filteredParams.map((p) => {
				if (p.Name === "title" || p.Name === "icon") {
					p.Required = true;
				}
				return p;
			});
			return adjustedParams;
		},
		apiData() {
			return {
				template: this.templateName,
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
				this.reset();
				this.templateName = "offline";
				this.resetTest();
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
	mounted() {
		this.$refs.modal.addEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal.addEventListener("hide.bs.modal", this.modalInvisible);
	},
	unmounted() {
		this.$refs.modal?.removeEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal?.removeEventListener("hide.bs.modal", this.modalInvisible);
	},
	methods: {
		reset() {
			this.values = { ...initialValues };
			this.resetTest();
		},
		async loadConfiguration() {
			try {
				const meter = (await api.get(`config/devices/meter/${this.id}`)).data.result;
				this.values = meter.config;
				this.applyDefaultsFromTemplate();
				this.templateName = this.values.template;
			} catch (e) {
				console.error(e);
			}
		},
		async loadProducts() {
			try {
				this.products = (await api.get("config/products/meter")).data.result;
			} catch (e) {
				console.error(e);
			}
		},
		async loadTemplate() {
			try {
				const opts = {
					params: {
						lang: this.$i18n.locale,
						name: this.templateName,
					},
				};
				this.template = (await api.get("config/templates/meter", opts)).data.result;
				this.applyDefaultsFromTemplate();
			} catch (e) {
				console.error(e);
			}
		},
		templateChanged() {
			this.reset();
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
				const success = await this.test(this.testMeter);
				if (!success) return;
				await sleep(250);
			}
			try {
				await api.post("config/devices/meter", this.apiData);
				this.$emit("meter-changed");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("create failed");
			}
		},
		async testManually() {
			await this.test(this.testMeter);
		},
		async testMeter() {
			let url = "config/test/meter";
			if (!this.isNew) {
				url += `/${this.id}`;
			}
			await api.post(url, this.apiData);
		},
		async update() {
			if (this.testUnknown) {
				const success = await this.test(this.testMeter);
				if (!success) return;
				await sleep(250);
			}
			try {
				await api.put(`config/devices/meter/${this.id}`, this.apiData);
				this.$emit("meter-changed");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("update failed");
			}
		},
		async remove() {
			try {
				await api.delete(`config/devices/meter/${this.id}`);
				this.$emit("meter-changed");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("delete failed");
			}
		},
		modalVisible() {
			this.isModalVisible = true;
		},
		modalInvisible() {
			this.isModalVisible = false;
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
</style>
