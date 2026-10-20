<template>
	<GenericModal
		id="siteModal"
		ref="modal"
		:title="$t('config.site.title')"
		data-testid="site-modal"
		config-modal-name="site"
		@open="open"
	>
		<p v-if="error" class="text-danger">{{ error }}</p>
		<form ref="form" class="container mx-0 px-0" @submit.prevent="save">
			<FormRow
				id="siteTitle"
				:label="$t('config.site.siteTitle.label')"
				:help="$t('config.site.siteTitle.description')"
			>
				<input id="siteTitle" v-model="title" class="form-control" />
			</FormRow>

			<FormRow
				id="siteCountry"
				:label="$t('config.site.country.label')"
				:help="$t('config.site.country.description')"
			>
				<select id="siteCountry" v-model="country" class="form-select">
					<option value="">---</option>
					<template v-if="browserCountry">
						<option :value="browserCountry.code">{{ browserCountry.name }}</option>
						<option disabled>─────</option>
					</template>
					<option v-for="c in countries" :key="c.code" :value="c.code">
						{{ c.name }}
					</option>
				</select>
			</FormRow>

			<FormRow
				id="siteCurrency"
				:label="$t('config.site.currency.label')"
				:help="$t('config.site.currency.description')"
				:example="currencyExample"
			>
				<select id="siteCurrency" v-model="currency" class="form-select" required>
					<option v-for="c in currencies" :key="c.code" :value="c.code">
						{{ c.code }} - {{ c.name }}
					</option>
				</select>
			</FormRow>

			<PropertyCollapsible>
				<template #advanced>
					<FormRow
						id="siteGridConnection"
						:label="$t('config.site.gridConnection.label')"
						:help="gridConnectionHelp"
					>
						<select
							id="siteGridConnection"
							v-model="gridConnection"
							class="form-select"
						>
							<option
								v-for="option in gridConnectionOptions"
								:key="option.value"
								:value="option.value"
							>
								{{ option.name }}
							</option>
						</select>
					</FormRow>
				</template>
			</PropertyCollapsible>

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

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import FormRow from "./FormRow.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import store from "@/store";
import api from "@/api";
import { COUNTRIES, CURRENCY, GRID_CONNECTION } from "@/types/evcc";
import formatter from "@/mixins/formatter";

export default defineComponent({
	name: "SiteModal",
	components: { FormRow, GenericModal, PropertyCollapsible },
	mixins: [formatter],
	emits: ["changed"],
	data() {
		return {
			saving: false,
			error: "",
			title: "",
			currency: CURRENCY.EUR,
			country: "",
			gridConnection: GRID_CONNECTION.THREE_PHASE_FOUR_WIRE,
			initial: {
				title: "",
				currency: CURRENCY.EUR as string,
				country: "",
				gridConnection: GRID_CONNECTION.THREE_PHASE_FOUR_WIRE,
			},
		};
	},
	computed: {
		currencies() {
			return Object.values(CURRENCY).map((code) => ({
				code,
				name: this.fmtCurrencyName(code),
			}));
		},
		countries() {
			return COUNTRIES.map((code) => ({ code, name: this.fmtCountryName(code) })).sort(
				(a, b) => a.name.localeCompare(b.name, this.$i18n.locale)
			);
		},
		browserCountry() {
			const region = new Intl.Locale(navigator.language).maximize().region;
			return this.countries.find((c) => c.code === region);
		},
		titleChanged() {
			return this.title !== this.initial.title;
		},
		currencyChanged() {
			return this.currency !== this.initial.currency;
		},
		countryChanged() {
			return this.country !== this.initial.country;
		},
		voltage() {
			return store.state?.voltage ?? 230;
		},
		lineVoltage() {
			return Math.round((this.voltage * Math.sqrt(3)) / 10) * 10;
		},
		gridConnectionOptions() {
			const { voltage, lineVoltage } = this;
			return [
				{
					value: GRID_CONNECTION.THREE_PHASE_FOUR_WIRE,
					name: this.$t("config.site.gridConnection.3p4w", { voltage, lineVoltage }),
				},
				{
					value: GRID_CONNECTION.THREE_PHASE_THREE_WIRE,
					name: this.$t("config.site.gridConnection.3p3w", { voltage }),
				},
			];
		},
		gridConnectionChanged() {
			return this.gridConnection !== this.initial.gridConnection;
		},
		gridConnectionHelp() {
			const { voltage, lineVoltage } = this;
			return this.$t(`config.site.gridConnection.description.${this.gridConnection}`, {
				voltage,
				lineVoltage,
			});
		},
		changed() {
			return (
				this.titleChanged ||
				this.currencyChanged ||
				this.countryChanged ||
				this.gridConnectionChanged
			);
		},
		currencyExample() {
			const price = this.fmtPricePerKWh(0.122, this.currency);
			const amount = this.fmtMoney(20.2, this.currency, true, true);
			return this.$t("config.site.currency.example", { price, amount });
		},
	},
	methods: {
		open() {
			this.saving = false;
			this.error = "";
			this.title = store.state?.siteTitle || "";
			this.currency = store.state?.currency || CURRENCY.EUR;
			this.country = store.state?.country || "";
			this.gridConnection =
				store.state?.gridConnection ?? GRID_CONNECTION.THREE_PHASE_FOUR_WIRE;
			this.initial = {
				title: this.title,
				currency: this.currency,
				country: this.country,
				gridConnection: this.gridConnection,
			};
		},
		async save() {
			this.saving = true;
			this.error = "";
			try {
				const requests = [];
				if (this.titleChanged) {
					requests.push(api.put("/config/site", { title: this.title }));
				}
				if (this.currencyChanged) {
					requests.push(api.put("/config/currency", JSON.stringify(this.currency)));
				}
				if (this.countryChanged) {
					requests.push(api.put("/config/country", JSON.stringify(this.country)));
				}
				if (this.gridConnectionChanged) {
					requests.push(
						api.put("/config/gridconnection", JSON.stringify(this.gridConnection))
					);
				}
				await Promise.all(requests);
				this.$emit("changed");
				const modalRef = this.$refs["modal"] as
					| InstanceType<typeof GenericModal>
					| undefined;
				modalRef?.close();
			} catch (e) {
				this.error = (e as Error).message;
			}
			this.saving = false;
		},
	},
});
</script>

<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
	padding-right: 0;
}
</style>
