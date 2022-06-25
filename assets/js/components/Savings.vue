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
								<strong>Zeitraum:</strong>
								{{
									$t("footer.savings.modalServerStart", {
										since: fmtTimeAgo(secondsSinceStart()),
									})
								}}
							</p>

							<div class="d-block d-lg-flex mb-4">
								<SavingsTile
									class="text-accent1"
									icon="sun"
									title="Sonnenenergie"
									:value="percent"
									unit="%"
									:sub1="`${fmtKw(
										selfConsumptionCharged * 1000,
										true,
										false
									)} kWh Sonne`"
									:sub2="`${fmtKw(gridCharged * 1000, true, false)} kWh Netz`"
								/>

								<SavingsTile
									class="text-accent2"
									icon="receivepayment"
									title="Ladeenergie"
									:value="fmtPricePerKWh(effectivePrice, currency).split(' ')[0]"
									:unit="fmtPricePerKWh(effectivePrice, currency).split(' ')[1]"
									:sub1="
										$t('footer.savings.modalExplainationFeedIn', {
											feedInPrice: fmtPricePerKWh(feedInPrice, currency),
										})
									"
									:sub2="
										$t('footer.savings.modalExplainationGrid', {
											gridPrice: fmtPricePerKWh(gridPrice, currency),
										})
									"
								/>

								<SavingsTile
									class="text-accent3"
									icon="coinjar"
									title="Ersparnis"
									:value="fmtMoney(amount, currency).split('&nbsp;')[0]"
									:unit="fmtMoney(amount, currency).split('&nbsp;')[1]"
									sub1="gegenüber Netzbezug"
									:sub2="`${fmtKw(totalCharged * 1000, true, false)} kWh geladen`"
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
		noData() {
			return this.totalCharged === 0;
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
