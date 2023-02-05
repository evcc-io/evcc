<template>
	<Teleport to="body">
		<div
			id="vehicleSettingsModal"
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
						<h5 class="modal-title">Add New Vehicle 🧪</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<div class="container">
							<FormRow id="settingsDesign" label="Icon">
								<input type="text" />
							</FormRow>
							<FormRow id="settingsDesign" label="Zugangsdaten">
								Volkswagen We Connect ID
								<div class="d-flex align-items-center">
									<small class="text-success"> Verbindung erfolgreich </small>
									<button class="btn-sm btn btn-link text-muted">
										bearbeiten
									</button>
								</div>
							</FormRow>
							<FormRow id="settingsLanguage" :label="$t('settings.language.label')">
								<select
									id="settingsLanguage"
									v-model="language"
									class="form-select form-select-sm w-75"
								>
									<option value="">{{ $t("settings.language.auto") }}</option>
								</select>
							</FormRow>
							<FormRow id="settingsUnit" :label="$t('settings.unit.label')">
								<SelectGroup
									id="settingsUnit"
									v-model="unit"
									class="w-75"
									:options="
										['1', '2'].map((value) => ({
											value,
											name: $t(`settings.unit.${value}`),
										}))
									"
								/>
							</FormRow>
						</div>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import FormRow from "./FormRow.vue";
import SelectGroup from "./SelectGroup.vue";

export default {
	name: "VehicleSettingsModal",
	components: { FormRow, SelectGroup },
	data() {
		return { isModalVisible: false };
	},
	watch: {
		isModalVisible(visible) {
			if (visible) {
				// daten laden
			}
		},
	},
	mounted() {
		this.$refs.modal.addEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal.addEventListener("hide.bs.modal", this.modalInvisible);
	},
	unmounted() {
		this.$refs.modal.removeEventListener("show.bs.modal", this.modalVisible);
		this.$refs.modal.removeEventListener("hide.bs.modal", this.modalInvisible);
	},
	methods: {
		modalVisible: function () {
			this.isModalVisible = true;
		},
		modalInvisible: function () {
			this.isModalVisible = false;
		},
	},
};
</script>
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
}

.container h4:first-child {
	margin-top: 0 !important;
}
</style>
