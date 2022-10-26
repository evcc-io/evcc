<template>
	<div>
		<button
			class="btn btn-link pe-0 text-decoration-none evcc-default-text text-nowrap d-flex align-items-end"
			data-bs-toggle="modal"
			data-bs-target="#savingsModal"
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
							<p>
								{{
									$t("footer.savings.since", {
										since: fmtTimeAgo(secondsSinceStart()),
									})
								}}
							</p>

							<div class="d-block d-lg-flex mb-4">
								<SavingsTile
									class="text-accent1"
									icon="sun"
									:title="$t('footer.savings.percentTitle')"
									:value="percent"
									unit="%"
									:sub1="
										$t('footer.savings.percentSelf', {
											self: fmtKw(selfConsumptionCharged * 1000, true, false),
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

							<Sponsor :sponsor="sponsor" />
						</div>
					</div>
				</div>
			</div>
		</Teleport>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";
import Sponsor from "./Sponsor.vue";
import SavingsTile from "./SavingsTile.vue";

export default {
	name: "Savings",
	components: { Sponsor, SavingsTile },
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
	},
	methods: {
		secondsSinceStart() {
			return this.since * 1000 - Date.now();
		},
	},
};
</script>
<style scoped>
.chart {
	height: 2.5rem;
}

.chart-item--self {
	background-color: var(--evcc-self);
}
.chart-item--grid {
	background-color: var(--evcc-grid);
}
.chart-item--no-data {
	background-color: var(--bs-gray-medium);
}

.chart-item {
	transition-property: width;
	transition-duration: var(--evcc-transition-medium);
	transition-timing-function: linear;
}
.tile-icon {
	width: 70px;
}
</style>
