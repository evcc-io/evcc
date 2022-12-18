<template>
	<div>
		<button
			class="btn btn-link pe-0 text-decoration-none evcc-default-text text-nowrap d-flex align-items-end"
			@click="openModal"
		>
			<span class="d-inline d-sm-none">{{
				$t("footer.savings.footerShort", { percent })
			}}</span
			><span class="d-none d-sm-inline">{{
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
				<div
					class="modal-dialog modal-lg modal-dialog-centered modal-dialog-scrollable"
					role="document"
				>
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
										:value="percent"
										unit="%"
										:sub1="
											$t('footer.savings.percentSelf', {
												self: fmtKw(
													selfConsumptionCharged * 1000,
													true,
													false
												),
											})
										"
										:sub2="
											$t('footer.savings.percentGrid', {
												grid: fmtKw(gridCharged * 1000, true, false),
											})
										"
									/>

									<SavingsTile
										class="text-accent2"
										icon="receivepayment"
										:title="$t('footer.savings.priceTitle')"
										:value="effectivePriceFormatted.value"
										:unit="effectivePriceFormatted.unit"
										:sub1="
											$t('footer.savings.priceFeedIn', {
												feedInPrice: fmtPricePerKWh(feedInPrice, currency),
											})
										"
										:sub2="
											$t('footer.savings.priceGrid', {
												gridPrice: fmtPricePerKWh(gridPrice, currency),
											})
										"
									/>

									<SavingsTile
										class="text-accent3"
										icon="coinjar"
										:title="$t('footer.savings.savingsTitle')"
										:value="fmtMoney(amount, currency)"
										:unit="fmtCurrencySymbol(currency)"
										:sub1="$t('footer.savings.savingsComparedToGrid')"
										:sub2="
											$t('footer.savings.savingsTotalEnergy', {
												total: fmtKw(totalCharged * 1000, true, false),
											})
										"
									/>
								</div>
								<p class="my-3 lh-2">
									<small>
										{{
											$t("footer.savings.since", {
												since: fmtDayMonthYear(startDate),
											})
										}}
									</small>
								</p>
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

export default {
	name: "Savings",
	components: { Sponsor, SavingsTile, LiveCommunity, TelemetrySettings },
	mixins: [formatter],
	props: {
		selfConsumptionPercent: Number,
		since: { type: Number, default: 0 },
		sponsor: String,
		amount: { type: Number, default: 0 },
		effectivePrice: { type: Number, default: 0 },
		totalCharged: { type: Number, default: 0 },
		gridCharged: { type: Number, default: 0 },
		selfConsumptionCharged: { type: Number, default: 0 },
		gridPrice: { type: Number },
		feedInPrice: { type: Number },
		currency: String,
	},
	data() {
		return { communityView: false, telemetryEnabled: false };
	},
	computed: {
		percent() {
			return Math.round(this.selfConsumptionPercent) || 0;
		},
		effectivePriceFormatted() {
			const [value, unit] = this.fmtPricePerKWh(this.effectivePrice, this.currency).split(
				" "
			);
			return { value, unit };
		},
		startDate() {
			return new Date(this.since * 1000);
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
	},
};
</script>
