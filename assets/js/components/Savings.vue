<template>
	<div>
		<button
			class="btn btn-link pe-0 text-decoration-none evcc-default-text text-nowrap d-flex align-items-end"
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
										:title="$t('footer.savings.percentTitle')"
										:value="fmtNumber(solarPercentage, 1)"
										unit="%"
										:sub1="
											$t('footer.savings.percentSelf', {
												self: fmtKw(solarCharged * 1000, true, false, 0),
											})
										"
										:sub2="
											$t('footer.savings.percentGrid', {
												grid: fmtKw(gridCharged * 1000, true, false, 0),
											})
										"
									/>

									<SavingsTile
										class="text-accent2"
										icon="receivepayment"
										:title="$t('footer.savings.priceTitle')"
										:value="priceConfigured ? avgPriceFormatted.value : '-'"
										:unit="avgPriceFormatted.unit"
										:sub1="
											region &&
											priceConfigured &&
											currency === region.currency
												? `${fmtMoney(
														(region.price - avgPrice) * totalCharged,
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
										:title="$t('footer.savings.co2Title')"
										:value="co2Configured ? fmtNumber(avgCo2, 0) : '-'"
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
										<div class="me-1">{{ $t("footer.savings.period") }}</div>
										<CustomSelect
											:selected="period"
											:options="periodOptions"
											data-testid="sessionInfoSelect"
											@change="selectPeriod($event.target.value)"
										>
											<span class="text-decoration-underline">
												{{ $t(`footer.savings.period${period}`) }}
											</span>
										</CustomSelect>
									</div>
									<div v-if="region" class="d-flex flex-wrap">
										<div class="me-1">{{ $t("footer.savings.reference") }}</div>
										<CustomSelect
											class="me-1"
											:selected="region.name"
											:options="regionOptions"
											data-testid="sessionInfoSelect"
											@change="selectRegion($event.target.value)"
										>
											<span class="text-decoration-underline">
												{{ region.name }}
											</span>
										</CustomSelect>
										<div class="evcc-gray">
											⌀
											{{ fmtPricePerKWh(region.price, region.currency) }}
											<a
												class="evcc-gray"
												:href="sources.price.url"
												target="_blank"
												>({{ $t("footer.savings.source") }})</a
											>
											, ⌀
											{{ fmtCo2Medium(region.co2) }}
											<a
												class="evcc-gray"
												:href="sources.co2.url"
												target="_blank"
												>({{ $t("footer.savings.source") }})</a
											>
										</div>
									</div>
									<div v-if="!priceConfigured || !co2Configured">
										<a
											href="https://docs.evcc.io/en/docs/reference/configuration/tariffs/"
											class="evcc-gray"
											target="_blank"
										>
											{{ $t("footer.savings.configurePriceCo2") }}
										</a>
									</div>
								</div>
							</div>
							<div v-else class="my-4">
								<LiveCommunity />
								<TelemetrySettings :sponsor="sponsor" />
							</div>

							<Sponsor :sponsor="sponsor" />
						</div>
					</div>
				</div>
			</div>
		</Teleport>
	</div>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import formatter from "../mixins/formatter";
import Sponsor from "./Sponsor.vue";
import SavingsTile from "./SavingsTile.vue";
import LiveCommunity from "./LiveCommunity.vue";
import TelemetrySettings from "./TelemetrySettings.vue";
import CustomSelect from "./CustomSelect.vue";
import referenceData from "../referenceData";
import settings from "../settings";

export default {
	name: "Savings",
	components: { Sponsor, SavingsTile, LiveCommunity, TelemetrySettings, CustomSelect },
	mixins: [formatter],
	props: {
		stats: { type: Object, default: () => ({}) },
		co2Configured: Boolean,
		priceConfigured: Boolean,
		sponsor: String,
		currency: String,
	},
	data() {
		return {
			communityView: false,
			telemetryEnabled: false,
			period: settings.savingsPeriod,
			selectedRegion: settings.savingsRegion,
			sources: referenceData.sources,
		};
	},
	computed: {
		percent() {
			return Math.round(this.solarPercentage) || 0;
		},
		regionOptions() {
			return referenceData.regions.map((r) => ({
				value: r.name,
				name: `${r.name} (${r.currency})`,
			}));
		},
		region() {
			// previously selected region
			if (this.selectedRegion) {
				const result = referenceData.regions.find((r) => r.name === this.selectedRegion);
				if (result) {
					return result;
				}
			}

			// if EUR and no selection, default to Germany
			if (this.currency === "EUR") {
				return referenceData.regions.find((r) => r.name === "Germany");
			}

			// region matching currency
			let result = referenceData.regions.find((r) => r.currency === this.currency);
			if (result) {
				return result;
			}

			// first region
			return referenceData.regions[0];
		},
		periodOptions() {
			return ["30d", "365d"].map((p) => ({
				value: p,
				name: this.$t(`footer.savings.period${p}`),
			}));
		},
		avgPriceFormatted() {
			const value = this.fmtPricePerKWh(
				this.currentStats.avgPrice,
				this.currency,
				false,
				false
			);
			const unit = this.pricePerKWhUnit(this.currency);
			return { value, unit };
		},
		range() {
			return this.$t(`footer.savings.period${this.period}`);
		},
		currentStats() {
			return this.stats[this.period] || {};
		},
		totalCharged() {
			return this.currentStats.chargedKWh;
		},
		solarPercentage() {
			return this.currentStats.solarPercentage;
		},
		solarCharged() {
			return (this.solarPercentage / 100) * this.totalCharged;
		},
		gridCharged() {
			return this.totalCharged - this.solarCharged;
		},
		avgPrice() {
			return this.currentStats.avgPrice;
		},
		avgCo2() {
			return this.currentStats.avgCo2;
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
			const modal = Modal.getOrCreateInstance(document.getElementById("savingsModal"));
			modal.show();
		},
		selectPeriod(period) {
			this.period = period;
			settings.savingsPeriod = period;
		},
		selectRegion(region) {
			this.selectedRegion = region;
			settings.savingsRegion = region;
		},
	},
};
</script>
