<template>
	<div>
		<button
			class="btn btn-link p-0 text-decoration-none text-nowrap d-flex align-items-center evcc-gray"
			data-testid="savings-button"
			@click="openModal"
		>
			<span v-if="indicatorValueShort" class="indicator-value d-block d-sm-none">{{
				indicatorValueShort
			}}</span>
			<span v-if="indicatorValue" class="indicator-value d-none d-sm-block">{{
				indicatorValue
			}}</span>
			<DynamicPriceIcon v-if="indicator === 'price'" class="ms-2" />
			<shopicon-regular-sun
				v-else-if="indicator === 'solar' || indicator === 'none'"
				class="ms-2"
			></shopicon-regular-sun>
			<shopicon-regular-receivepayment
				v-else-if="indicator === 'savings'"
				class="ms-2"
			></shopicon-regular-receivepayment>
			<shopicon-regular-eco1 v-else class="ms-2"></shopicon-regular-eco1>
		</button>

		<Teleport to="body">
			<div
				id="savingsModal"
				ref="modal"
				class="modal fade text-dark"
				data-bs-backdrop="true"
				tabindex="-1"
				role="dialog"
				aria-hidden="true"
				data-testid="savings-modal"
			>
				<div class="modal-dialog modal-lg modal-dialog-centered" role="document">
					<div class="modal-content">
						<div class="modal-header">
							<h5 class="modal-title">
								{{ $t("footer.savings.modalTitle") }}
							</h5>
							<button
								type="button"
								class="btn-close"
								data-bs-dismiss="modal"
								aria-label="Close"
							></button>
						</div>
						<div class="modal-body">
							<ul class="nav nav-tabs">
								<li class="nav-item">
									<a
										class="nav-link"
										:class="{ active: !communityView }"
										href="#"
										@click.prevent="showMyData"
									>
										{{ $t("footer.savings.tabTitle") }}
									</a>
								</li>
								<li class="nav-item">
									<a
										class="nav-link"
										:class="{ active: communityView }"
										href="#"
										@click.prevent="showCommunity"
									>
										{{ $t("footer.community.tabTitle") }}
									</a>
								</li>
							</ul>

							<div v-if="!communityView" class="my-4">
								<div class="d-block d-lg-flex mb-2 justify-content-between">
									<SavingsTile
										class="text-accent1"
										icon="sun"
										data-testid="savings-tile-solar"
										:title="$t('footer.savings.percentTitle')"
										:value="fmtNumber(solarPercentage, 1)"
										unit="%"
										:sub1="
											$t('footer.savings.percentSelf', {
												self: fmtW(
													solarCharged * 1000,
													POWER_UNIT.KW,
													false,
													0
												),
											})
										"
										:sub2="
											$t('footer.savings.percentGrid', {
												grid: fmtW(
													gridCharged * 1000,
													POWER_UNIT.KW,
													false,
													0
												),
											})
										"
									/>

									<SavingsTile
										class="text-accent2"
										icon="receivepayment"
										data-testid="savings-tile-price"
										:title="$t('footer.savings.priceTitle')"
										:value="priceConfigured ? avgPriceFormatted.value : '__'"
										:unit="avgPriceFormatted.unit"
										:sub1="
											priceConfigured && referenceGrid
												? `${fmtMoney(
														moneySaved,
														currency,
														false
													)} ${fmtCurrencySymbol(currency)} ${$t(
														'footer.savings.moneySaved'
													)}`
												: ''
										"
									/>

									<SavingsTile
										class="text-accent3"
										icon="eco"
										data-testid="savings-tile-co2"
										:title="$t('footer.savings.co2Title')"
										:value="co2Configured ? fmtNumber(avgCo2, 0) : '__'"
										unit="g/kWh"
										:sub1="
											region && co2Configured
												? `${fmtNumber(
														co2Saved,
														0,
														'kilogram'
													)} ${$t('footer.savings.co2Saved')}`
												: ''
										"
									/>
								</div>
								<table class="mt-3 mb-2 lh-2">
									<tbody>
										<tr>
											<td class="pe-3 align-top">
												{{ $t("footer.savings.periodLabel") }}
											</td>
											<td>
												<CustomSelect
													id="savingsPeriod"
													:selected="period"
													:options="periodOptions"
													data-testid="savings-period-select"
													@change="selectPeriod($event.target.value)"
												>
													<span
														class="text-decoration-underline evcc-gray"
													>
														{{ $t(`footer.savings.period.${period}`) }}
													</span>
												</CustomSelect>
											</td>
										</tr>
										<tr>
											<td class="pe-3 align-top">
												{{ $t("footer.savings.indicatorLabel") }}
											</td>
											<td>
												<CustomSelect
													:selected="indicator"
													:options="indicatorOptions"
													data-testid="savings-indicator-select"
													@change="selectIndicator($event.target.value)"
												>
													<span
														class="text-decoration-underline evcc-gray"
													>
														{{
															indicatorValue
																? `${indicatorValue} ${$t(`footer.savings.indicator.${indicator}`)}`
																: $t(
																		`footer.savings.indicator.${indicator}`
																	)
														}}
													</span>
												</CustomSelect>
											</td>
										</tr>
										<tr v-if="region" data-testid="savings-reference">
											<td class="pe-3 align-top">
												{{ $t("footer.savings.referenceLabel") }}
											</td>
											<td class="evcc-gray">
												<div>
													<span v-if="isDynamicPrice">⌀ </span
													>{{
														priceConfigured
															? fmtPricePerKWh(
																	referenceGrid,
																	currency
																)
															: "___"
													}}
													(<a
														href="#"
														class="evcc-gray text-decoration-underline"
														@click.prevent="navigateToTariffs"
														>{{ $t("config.main.title") }}</a
													>)
												</div>
												<div class="d-flex">
													<span class="me-1"
														>⌀ {{ fmtCo2Medium(region.co2) }}</span
													>
													<CustomSelect
														class="evcc-gray"
														:selected="region.name"
														:options="regionOptions"
														data-testid="savings-region-select"
														@change="selectRegion($event.target.value)"
													>
														(<span class="text-decoration-underline">{{
															region.name
														}}</span
														>)
													</CustomSelect>
												</div>
											</td>
										</tr>
									</tbody>
								</table>
								<div v-if="!priceConfigured || !co2Configured">
									<a
										href="#"
										class="evcc-gray"
										@click.prevent="navigateToTariffs"
									>
										{{ $t("footer.savings.configurePriceCo2") }}
									</a>
								</div>
								<div class="evcc-gray small">
									{{ $t("footer.savings.sessionInfo") }}
								</div>
							</div>
							<div v-else class="my-4">
								<LiveCommunity />
								<TelemetrySettings
									:sponsorActive="sponsorActive"
									:telemetry="telemetry"
								/>
							</div>
							<Sponsor v-bind="sponsor" />
						</div>
					</div>
				</div>
			</div>
		</Teleport>
	</div>
</template>

<script lang="ts">
import Modal from "bootstrap/js/dist/modal";
import formatter from "@/mixins/formatter";
import Sponsor from "./Sponsor.vue";
import Tile from "./Tile.vue";
import LiveCommunity from "./LiveCommunity.vue";
import TelemetrySettings from "../TelemetrySettings.vue";
import CustomSelect from "../Helper/CustomSelect.vue";
import DynamicPriceIcon from "../MaterialIcon/DynamicPrice.vue";
import "@h2d2/shopicons/es/regular/receivepayment";
import "@h2d2/shopicons/es/regular/eco1";
import settings from "@/settings.ts";
import { defineComponent, type PropType } from "vue";
import type {
	CURRENCY,
	Forecast,
	StatisticsIndicator,
	StatisticsPeriod,
	SelectOption,
	Sponsor as SponsorType,
} from "@/types/evcc";
import co2Reference from "./co2Reference.ts";

export default defineComponent({
	name: "Savings",
	components: {
		Sponsor,
		SavingsTile: Tile,
		LiveCommunity,
		TelemetrySettings,
		CustomSelect,
		DynamicPriceIcon,
	},
	mixins: [formatter],
	props: {
		statistics: { type: Object, default: () => ({}) },
		co2Configured: Boolean,
		sponsor: Object as PropType<SponsorType>,
		currency: String as PropType<CURRENCY>,
		telemetry: Boolean,
		forecast: Object as PropType<Forecast>,
		tariffGrid: Number,
	},
	data() {
		return {
			communityView: false,
			telemetryEnabled: false,
			period: settings.savingsPeriod || ("30d" as StatisticsPeriod),
			selectedRegion: settings.savingsRegion || ("Germany" as string),
		};
	},
	computed: {
		sponsorActive(): boolean {
			return !!this.sponsor?.status?.name;
		},
		percent() {
			return this.fmtPercentage(this.solarPercentage || 0);
		},
		regionOptions() {
			return co2Reference.regions.map((r) => ({
				value: r.name,
				name: `${r.name} (${this.fmtCo2Short(r.co2)})`,
			}));
		},
		region() {
			// previously selected region
			if (this.selectedRegion) {
				const result = co2Reference.regions.find((r) => r.name === this.selectedRegion);
				if (result) {
					return result;
				}
			}

			// first region
			return co2Reference.regions[0];
		},
		periodOptions(): SelectOption<StatisticsPeriod>[] {
			return (["30d", "365d", "thisYear", "total"] as StatisticsPeriod[]).map((p) => ({
				value: p,
				name: this.$t(`footer.savings.period.${p}`),
			}));
		},
		avgPriceFormatted() {
			const value = this.fmtPricePerKWh(
				this.currentStatistics.avgPrice,
				this.currency,
				false,
				false
			);
			const unit = this.pricePerKWhUnit(this.currency);
			return { value, unit };
		},
		currentStatistics() {
			return this.statistics[this.period] || {};
		},
		totalCharged() {
			return this.currentStatistics.chargedKWh;
		},
		solarPercentage() {
			return this.currentStatistics.solarPercentage;
		},
		solarCharged() {
			return (this.solarPercentage / 100) * this.totalCharged;
		},
		gridCharged() {
			return this.totalCharged - this.solarCharged;
		},
		avgPrice() {
			return this.currentStatistics.avgPrice;
		},
		avgCo2() {
			return this.currentStatistics.avgCo2;
		},
		referenceGrid(): number | undefined {
			const grid = this.forecast?.grid;
			if (grid?.length) {
				return grid.reduce((acc, slot) => acc + slot.value, 0) / grid.length;
			}
			return this.tariffGrid;
		},
		isDynamicPrice(): boolean {
			const grid = this.forecast?.grid;
			if (!grid?.length) return false;
			return grid.some((slot) => slot.value !== grid[0].value);
		},
		priceConfigured() {
			return this.referenceGrid !== undefined;
		},
		moneySaved(): number {
			return Math.max(0, ((this.referenceGrid ?? 0) - this.avgPrice) * this.totalCharged);
		},
		co2Saved(): number {
			return Math.max(
				0,
				(((this.region?.co2 ?? 0) - this.avgCo2) * this.totalCharged) / 1000
			);
		},
		indicator(): StatisticsIndicator {
			return (settings.savingsIndicator as StatisticsIndicator) || "solar";
		},
		indicatorOptions(): SelectOption<StatisticsIndicator>[] {
			return (
				["none", "solar", "price", "savings", "co2", "co2saved"] as StatisticsIndicator[]
			).map((key) => {
				const label = this.$t(`footer.savings.indicator.${key}`);
				const val = this.indicatorValueFor(key);
				return {
					value: key,
					name: val ? `${val} ${label}` : label,
				};
			});
		},
		indicatorValue(): string | undefined {
			return this.indicatorValueFor(this.indicator);
		},
		indicatorValueShort(): string | undefined {
			return this.indicatorValueFor(this.indicator, true);
		},
	},
	methods: {
		showCommunity() {
			this.communityView = true;
		},
		showMyData() {
			this.communityView = false;
		},
		openModal() {
			const modal = Modal.getOrCreateInstance(this.$refs["modal"] as HTMLElement);
			modal.show();
		},
		selectPeriod(period: StatisticsPeriod) {
			this.period = period;
			settings.savingsPeriod = period;
		},
		selectRegion(region: string) {
			this.selectedRegion = region;
			settings.savingsRegion = region;
		},
		selectIndicator(value: StatisticsIndicator) {
			settings.savingsIndicator = value;
		},
		navigateToTariffs() {
			const modal = Modal.getInstance(this.$refs["modal"] as HTMLElement);
			modal?.hide();
			this.$router.push("/config#tariffs");
		},
		indicatorValueFor(key: StatisticsIndicator, short = false): string | undefined {
			switch (key) {
				case "solar":
					return this.percent;
				case "price":
					if (this.avgPrice === undefined || !this.priceConfigured) return undefined;
					return this.fmtPricePerKWh(this.avgPrice, this.currency, short);
				case "savings":
					if (!this.priceConfigured || this.referenceGrid === undefined) return undefined;
					return `${this.fmtMoney(this.moneySaved, this.currency)} ${this.fmtCurrencySymbol(this.currency)}`;
				case "co2":
					if (this.avgCo2 === undefined || !this.co2Configured) return undefined;
					return short ? this.fmtCo2Short(this.avgCo2) : this.fmtCo2Medium(this.avgCo2);
				case "co2saved":
					if (!this.co2Configured || !this.region) return undefined;
					return `${this.fmtNumber(this.co2Saved, 0, "kilogram")}`;
				default:
					return undefined;
			}
		},
	},
});
</script>

<style scoped>
.indicator-value {
	font-size: 1rem;
}
</style>
