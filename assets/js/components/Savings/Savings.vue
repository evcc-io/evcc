<template>
	<div>
		<button
			class="btn btn-link pe-0 text-decoration-none evcc-default-text text-nowrap d-flex align-items-end"
			data-testid="savings-button"
			@click="openModal"
		>
			<span class="d-inline d-sm-none text-decoration-underline">{{
				$t("footer.savings.footerShort", { percent })
			}}</span
			><span class="d-none d-sm-inline text-decoration-underline">{{
				$t("footer.savings.footerLong", { percent })
			}}</span>
			<shopicon-regular-sun class="ms-2 text-evcc"></shopicon-regular-sun>
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
														(referenceGrid - avgPrice) * totalCharged,
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
														((region.co2 - avgCo2) * totalCharged) /
															1000,
														0,
														'kilogram'
													)} ${$t('footer.savings.co2Saved')}`
												: ''
										"
									/>
								</div>
								<div class="my-3 lh-2">
									<div class="d-flex">
										<label for="savingsPeriod" class="me-1">
											{{ $t("footer.savings.periodLabel") }}
										</label>
										<CustomSelect
											id="savingsPeriod"
											:selected="period"
											:options="periodOptions"
											data-testid="savings-period-select"
											@change="selectPeriod($event.target.value)"
										>
											<span class="text-decoration-underline evcc-gray">
												{{ $t(`footer.savings.period.${period}`) }}
											</span>
										</CustomSelect>
									</div>
									<div
										v-if="region"
										class="d-flex flex-wrap"
										data-testid="savings-reference"
									>
										<div class="me-1">
											{{ $t("footer.savings.referenceLabel") }}
										</div>
										<div class="evcc-gray me-1">
											{{
												priceConfigured
													? fmtPricePerKWh(referenceGrid, currency)
													: "___"
											}}
											({{ $t("footer.savings.referenceGrid") }}),
										</div>
										<div class="evcc-gray d-flex">
											<div class="me-1">⌀ {{ fmtCo2Medium(region.co2) }}</div>
											<CustomSelect
												class="me-1 evcc-gray"
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
									</div>
									<div v-if="!priceConfigured || !co2Configured">
										<a :href="tariffLink" class="evcc-gray" target="_blank">
											{{ $t("footer.savings.configurePriceCo2") }}
										</a>
									</div>
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
import settings from "@/settings.ts";
import api, { allowClientError } from "@/api.ts";
import { docsPrefix } from "@/i18n.ts";
import { defineComponent, type PropType } from "vue";
import type { CURRENCY, Rate, SelectOption, Sponsor as SponsorType } from "@/types/evcc";
import type { Period } from "./types";
import co2Reference from "./co2Reference.ts";

export default defineComponent({
	name: "Savings",
	components: { Sponsor, SavingsTile: Tile, LiveCommunity, TelemetrySettings, CustomSelect },
	mixins: [formatter],
	props: {
		statistics: { type: Object, default: () => ({}) },
		co2Configured: Boolean,
		sponsor: Object as PropType<SponsorType>,
		currency: String as PropType<CURRENCY>,
		telemetry: Boolean,
	},
	data() {
		return {
			communityView: false,
			telemetryEnabled: false,
			period: settings.savingsPeriod || ("30d" as Period),
			selectedRegion: settings.savingsRegion || ("Germany" as string),
			referenceGrid: undefined as number | undefined,
		};
	},
	computed: {
		sponsorActive(): boolean {
			return !!this.sponsor?.status.name;
		},
		tariffLink() {
			return `${docsPrefix()}/docs/reference/configuration/tariffs`;
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
		periodOptions(): SelectOption<Period>[] {
			return (["30d", "365d", "thisYear", "total"] as Period[]).map((p) => ({
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
		priceConfigured() {
			return this.referenceGrid !== undefined;
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
			const modal = Modal.getOrCreateInstance(
				document.getElementById("savingsModal") as HTMLElement
			);
			modal.show();
			this.updateReferenceGrid();
		},
		selectPeriod(period: Period) {
			this.period = period;
			settings.savingsPeriod = period;
		},
		selectRegion(region: string) {
			this.selectedRegion = region;
			settings.savingsRegion = region;
		},
		async updateReferenceGrid() {
			try {
				const res = await api.get(`tariff/grid`, allowClientError);
				const { rates } = res.data as { rates: Rate[] };
				this.referenceGrid =
					rates.reduce((acc, slot) => {
						return acc + slot.value;
					}, 0) / rates.length;
			} catch (e) {
				this.referenceGrid = undefined;
				console.error(e);
			}
		},
	},
});
</script>
