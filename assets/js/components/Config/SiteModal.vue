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
				:label="$t('config.site.sitetitle.label')"
				:help="$t('config.site.sitetitle.description')"
			>
				<input id="siteTitle" v-model="title" class="form-control" />
			</FormRow>

			<FormRow
				id="currency"
				:label="$t('config.site.currency.label')"
				:example="exampleText"
				:help="$t('config.site.currency.description')"
			>
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

			<h5 class="mt-4 mb-3">{{ $t("config.site.geolocation.title") }}</h5>
			<p class="text-muted small">{{ $t("config.site.geolocation.description") }}</p>
			<div class="mb-4">
				<div class="form-check form-switch">
					<input
						id="geoLocationEnabled"
						v-model="geoLocationEnabled"
						type="checkbox"
						class="form-check-input"
					/>
					<label class="form-check-label ms-3" for="geoLocationEnabled">
						{{ $t("config.site.geolocation.enabled.label") }}
					</label>
				</div>
			</div>

			<FormRow
				v-if="geoLocationEnabled"
				id="geoLocationLatitude"
				:label="$t('config.site.geolocation.latitude.label')"
				:example="$t('config.site.geolocation.latitude.example')"
				:help="$t('config.site.geolocation.latitude.description')"
			>
				<input
					id="geoLocationLatitude"
					v-model.number="geoLocationLatitude"
					type="number"
					step="0.00001"
					min="-90"
					max="90"
					class="form-control"
				/>
			</FormRow>

			<FormRow
				v-if="geoLocationEnabled"
				id="geoLocationLongitude"
				:label="$t('config.site.geolocation.longitude.label')"
				:example="$t('config.site.geolocation.longitude.example')"
				:help="$t('config.site.geolocation.longitude.description')"
			>
				<input
					id="geoLocationLongitude"
					v-model.number="geoLocationLongitude"
					type="number"
					step="0.00001"
					min="-180"
					max="180"
					class="form-control"
				/>
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
					>
					</span>
					{{ $t("config.general.save") }}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script lang="ts">
import GenericModal from "../Helper/GenericModal.vue";
import FormRow from "./FormRow.vue";
import store from "@/store";
import api from "@/api";
import { CURRENCY } from "@/types/evcc";
import formatter from "@/mixins/formatter";

export default {
	name: "SiteModal",
	components: { FormRow, GenericModal },
	mixins: [formatter],
	emits: ["changed"],
	data() {
		return {
			saving: false,
			error: "",
			selectedCurrency: CURRENCY.EUR,
			initialCurrency: CURRENCY.EUR,
			title: "",
			initialTitle: "",
			geoLocationEnabled: false,
			initialGeoLocationEnabled: false,
			geoLocationLatitude: 0,
			initialGeoLocationLatitude: 0,
			geoLocationLongitude: 0,
			initialGeoLocationLongitude: 0,
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
			return (
				this.title !== this.initialTitle ||
				this.selectedCurrency !== this.initialCurrency ||
				this.geoLocationEnabled !== this.initialGeoLocationEnabled ||
				this.geoLocationLatitude !== this.initialGeoLocationLatitude ||
				this.geoLocationLongitude !== this.initialGeoLocationLongitude
			);
		},
		exampleText() {
			const price = this.fmtPricePerKWh(0.122, this.selectedCurrency);
			const amount = this.fmtMoney(20.2, this.selectedCurrency, true, true);
			return this.$t("config.site.currency.example", { price, amount });
		},
	},
	methods: {
		reset() {
			const currency = store?.state?.currency || "EUR";
			const geoLocation: any = store?.state?.geoLocation || {};
			this.saving = false;
			this.error = "";
			this.selectedCurrency = currency as CURRENCY;
			this.initialCurrency = currency as CURRENCY;
			this.title = store.state?.siteTitle || "";
			this.initialTitle = this.title;
			this.geoLocationEnabled = geoLocation.enabled || false;
			this.initialGeoLocationEnabled = this.geoLocationEnabled;
			this.geoLocationLatitude = geoLocation.lat || 0;
			this.initialGeoLocationLatitude = this.geoLocationLatitude;
			this.geoLocationLongitude = geoLocation.lon || 0;
			this.initialGeoLocationLongitude = this.geoLocationLongitude;
		},
		async open() {
			this.reset();
		},
		async save() {
			this.saving = true;
			this.error = "";
			try {
				const requests = [];
				if (this.title !== this.initialTitle) {
					requests.push(api.put("/config/site", { title: this.title }));
				}
				if (this.selectedCurrency !== this.initialCurrency) {
					requests.push(
						api.put("/config/currency", JSON.stringify(this.selectedCurrency))
					);
				}
				if (
					this.geoLocationEnabled !== this.initialGeoLocationEnabled ||
					this.geoLocationLatitude !== this.initialGeoLocationLatitude ||
					this.geoLocationLongitude !== this.initialGeoLocationLongitude
				) {
					requests.push(
						api.put("/config/site", {
							geoLocation: {
								enabled: this.geoLocationEnabled,
								lat: this.geoLocationLatitude,
								lon: this.geoLocationLongitude,
							},
						})
					);
				}
				await Promise.all(requests);
				this.$emit("changed");
				(this.$refs as any)["modal"].close();
			} catch (e) {
				this.error = (e && (e as any).message) || String(e);
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
