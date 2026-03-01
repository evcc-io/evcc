<template>
	<GenericModal
		id="currencyModal"
		ref="modal"
		:title="$t('config.currency.title')"
		data-testid="currency-modal"
		config-modal-name="currency"
		@open="open"
	>
		<p>{{ $t("config.currency.description") }}</p>
		<p v-if="error" class="text-danger">{{ error }}</p>
		<form ref="form" class="container mx-0 px-0" @submit.prevent="save">
			<FormRow id="currency" :label="$t('config.currency.label')" :example="exampleText">
				<select id="currency" v-model="selectedCurrency" class="form-select" required>
					<option
						v-for="currency in currencies"
						:key="currency.code"
						:value="currency.code"
					>
						{{ currency.code }} - {{ currency.name }}
					</option>
				</select>
			</FormRow>

			<div class="mt-4 d-flex justify-content-between gap-2 flex-column flex-sm-row">
				<button
					type="button"
					class="btn btn-link text-muted btn-cancel"
					data-bs-dismiss="modal"
				>
					{{ $t("config.general.cancel") }}
				</button>

				<button
					type="submit"
					class="btn btn-primary order-1 order-sm-2 flex-grow-1 flex-sm-grow-0 px-4"
					:disabled="saving || !changed"
				>
					<span
						v-if="saving"
						class="spinner-border spinner-border-sm"
						role="status"
						aria-hidden="true"
					></span>
					{{ $t("config.general.save") }}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script>
import GenericModal from "../Helper/GenericModal.vue";
import FormRow from "./FormRow.vue";
import store from "@/store";
import api from "@/api";
import { CURRENCY } from "@/types/evcc";
import formatter from "@/mixins/formatter";

export default {
	name: "CurrencyModal",
	components: { FormRow, GenericModal },
	mixins: [formatter],
	emits: ["changed"],
	data() {
		return {
			saving: false,
			error: "",
			selectedCurrency: "EUR",
			initialCurrency: "EUR",
		};
	},
	computed: {
		currencies() {
			return Object.values(CURRENCY).map((code) => ({
				code,
				name: this.fmtCurrencyName(code),
			}));
		},
		changed() {
			return this.selectedCurrency !== this.initialCurrency;
		},
		exampleText() {
			const price = this.fmtPricePerKWh(0.122, this.selectedCurrency);
			const amount = this.fmtMoney(20.2, this.selectedCurrency, true, true);
			return this.$t("config.currency.example", { price, amount });
		},
	},
	methods: {
		reset() {
			const currency = store?.state?.currency || "EUR";
			this.saving = false;
			this.error = "";
			this.selectedCurrency = currency;
			this.initialCurrency = currency;
		},
		async open() {
			this.reset();
		},
		async save() {
			this.saving = true;
			this.error = "";
			try {
				await api.put("/config/currency", JSON.stringify(this.selectedCurrency));
				this.$emit("changed");
				this.$refs.modal.close();
			} catch (e) {
				this.error = e.message;
			}
			this.saving = false;
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
