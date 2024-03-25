<template>
	<Teleport to="body">
		<div
			id="loadpointModal"
			ref="modal"
			class="modal fade text-dark"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
			data-testid="loadpoint-modal"
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
							<FormRow
								v-for="param in formFields"
								:id="`meterParam${param.Name}`"
								:key="param.Name"
								:optional="!param.Required"
								:label="param.Description || `[${param.Name}]`"
								:help="param.Description === param.Help ? undefined : param.Help"
								:example="param.Example"
							>
								<PropertyField
									:id="`meterParam${param.Name}`"
									v-model="values[param.Name]"
									:masked="param.Mask"
									:property="param.Name"
									:type="param.Type"
									class="me-2"
									:required="param.Required"
									:validValues="param.ValidValues"
								/>
							</FormRow>

							<div class="my-4 d-flex justify-content-between">
								<button
									v-if="isDeletable"
									type="button"
									class="btn btn-link text-danger"
									@click.prevent="remove"
								>
									{{ $t("config.loadpoint.delete") }}
								</button>
								<button
									v-else
									type="button"
									class="btn btn-link text-muted"
									data-bs-dismiss="modal"
								>
									{{ $t("config.loadpoint.cancel") }}
								</button>
								<button
									type="submit"
									class="btn btn-primary"
									:disabled="saving"
									@click.prevent="isNew ? create() : update()"
								>
									<span
										v-if="saving"
										class="spinner-border spinner-border-sm"
										role="status"
										aria-hidden="true"
									></span>
									{{ $t("config.loadpoint.save") }}
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
import api from "../../api";

function sleep(ms) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

const priorityValues = Array.from({ length: 11 }, (_, i) => ({ key: i, name: `${i}` }));
priorityValues[0].name = "0 (default)";
priorityValues[10].name = "10 (highest)";

const formFields = [
	{
		Name: "title",
		Description: "Title",
		Example: "Garage, Carport, etc.",
		Type: "String",
		Required: true,
	},
	{
		Name: "phases",
		Description: "Phases",
		Help: "Electrical connection of the charger.",
		ValidValues: [
			{ key: 0, name: "automatic switching" },
			{ key: 1, name: "1-phase" },
			{ key: 2, name: "2-phase" },
			{ key: 3, name: "3-phase" },
		],
		Type: "Number",
		Required: true,
	},
	{
		Name: "minCurrent",
		Description: "Min Current",
		Example: "6A ... 32A",
		Type: "Number",
		Unit: "A",
		Required: true,
	},
	{
		Name: "maxCurrent",
		Description: "Max Current",
		Example: "6A ... 32A",
		Type: "Number",
		Unit: "A",
		Required: true,
	},
	{
		Name: "priority",
		Description: "Priority",
		Help: "Higher value means preferred charging with solar surplus.",
		Type: "Number",
		ValidValues: priorityValues,
		Required: true,
	},
];

/*
id: 1;
maxCurrent: 16;
minCurrent: 6;
mode: "now";
phases: 3;
priority: 0;
smartCostLimit: 0;
title: "Garage";
*/

export default {
	name: "MeterModal",
	components: { FormRow, PropertyField },
	props: {
		id: Number,
		name: String,
	},
	emits: ["added", "updated", "removed"],
	data() {
		return {
			isModalVisible: false,
			saving: false,
			selectedType: null,
			formFields,
			values: {},
		};
	},
	computed: {
		modalTitle() {
			if (this.isNew) {
				return this.$t(`config.loadpoint.titleAdd`);
			}
			return this.$t(`config.loadpoint.titleEdit`);
		},
		apiData() {
			// only send the fields that are present in the form
			return formFields.reduce((acc, field) => {
				acc[field.Name] = this.values[field.Name];
				return acc;
			}, {});
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
				if (this.id !== undefined) {
					this.loadConfiguration();
				}
			}
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
			this.values = {};
		},
		async loadConfiguration() {
			try {
				this.values = (await api.get(`config/loadpoints/${this.id}`)).data.result;
			} catch (e) {
				console.error(e);
			}
		},
		async update() {
			this.saving = true;
			try {
				await api.put(`config/loadpoints/${this.id}`, this.apiData);
				this.$emit("updated");
				this.modalInvisible();
			} catch (e) {
				console.error(e);
				alert("update failed");
			}
			this.saving = false;
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
.addButton {
	min-height: auto;
}
</style>
