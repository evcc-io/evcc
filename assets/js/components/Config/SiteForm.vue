<template>
	<form :id="`${idPrefix}Form`" class="container mx-0 px-0" @submit.prevent="save">
		<p v-if="error" class="text-danger">{{ error }}</p>
		<component
			:is="row"
			:id="`${idPrefix}Title`"
			:label="$t('config.site.siteTitle.label')"
			:help="help('siteTitle')"
		>
			<input :id="`${idPrefix}Title`" v-model="title" class="form-control" />
		</component>

		<component
			:is="row"
			:id="`${idPrefix}Country`"
			:label="$t('config.site.country.label')"
			:help="help('country')"
		>
			<select :id="`${idPrefix}Country`" v-model="country" class="form-select">
				<option value="">---</option>
				<template v-if="browserCountry">
					<option :value="browserCountry.code">{{ browserCountry.name }}</option>
					<option disabled>─────</option>
				</template>
				<option v-for="c in countries" :key="c.code" :value="c.code">
					{{ c.name }}
				</option>
			</select>
		</component>

		<component
			:is="row"
			:id="`${idPrefix}Currency`"
			:label="$t('config.site.currency.label')"
			:help="help('currency')"
			:example="currencyExample"
		>
			<select :id="`${idPrefix}Currency`" v-model="currency" class="form-select" required>
				<option v-for="c in currencies" :key="c.code" :value="c.code">
					{{ c.code }} - {{ c.name }}
				</option>
			</select>
			<div v-if="compact" class="form-text evcc-gray">{{ currencyExample }}</div>
		</component>

		<PropertyCollapsible v-if="!compact">
			<template #advanced>
				<FormRow
					:id="`${idPrefix}GridConnection`"
					:label="$t('config.site.gridConnection.label')"
					:help="gridConnectionHelp"
				>
					<select
						:id="`${idPrefix}GridConnection`"
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

		<slot :saving="saving" :changed="changed" />
	</form>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import FormRow from "./FormRow.vue";
import CompactFormRow from "../Helper/FormRow.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import store from "@/store";
import api from "@/api";
import { COUNTRIES, CURRENCY, GRID_CONNECTION } from "@/types/evcc";
import formatter from "@/mixins/formatter";

export default defineComponent({
	name: "SiteForm",
	components: { FormRow, CompactFormRow, PropertyCollapsible },
	mixins: [formatter],
	props: {
		// title, country and currency only
		compact: Boolean,
		// dom ids, the form is rendered in the site modal and in the setup wizard
		idPrefix: { type: String, default: "site" },
	},
	emits: ["saved"],
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
		// the wizard uses the label-left row without help texts
		row() {
			return this.compact ? "CompactFormRow" : "FormRow";
		},
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
			const key = this.compact
				? "onboarding.site.currencyHint"
				: "config.site.currency.example";
			return this.$t(key, { price, amount });
		},
	},
	mounted() {
		this.load();
	},
	methods: {
		help(key: string): string | undefined {
			return this.compact ? undefined : this.$t(`config.site.${key}.description`);
		},
		load() {
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
			if (this.compact) {
				if (!this.title) this.title = this.$t("onboarding.site.defaultTitle");
				if (!this.country) this.country = this.browserCountry?.code || "";
			}
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
				this.$emit("saved");
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
