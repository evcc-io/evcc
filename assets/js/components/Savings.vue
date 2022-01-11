<template>
	<div>
		<button
			class="btn btn-link pe-0 text-decoration-none link-dark text-nowrap"
			data-bs-toggle="modal"
			data-bs-target="#savingsModal"
		>
			<span class="d-inline d-sm-none">{{
				$t("footer.savings.footerShort", { percent })
			}}</span
			><span class="d-none d-sm-inline">{{
				$t("footer.savings.footerLong", { percent })
			}}</span
			><fa-icon icon="sun" class="icon ms-2 text-evcc"></fa-icon>
		</button>
		<div
			id="savingsModal"
			ref="modal"
			class="modal fade"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							{{
								$t("footer.savings.modalTitle", {
									total: fmtKw(chargedTotal * 1000, true, false),
								})
							}}
						</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body py-4">
						<div class="chart-container mb-3">
							<div class="chart-legend d-flex flex-wrap justify-content-between mb-1">
								<div class="text-nowrap">
									<fa-icon icon="square" class="text-evcc"></fa-icon>
									{{
										$t("footer.savings.modalChartSelf", {
											self: fmtKw(chargedSelfConsumption * 1000, true, false),
										})
									}}
								</div>
								<div class="text-nowrap">
									<fa-icon icon="square" class="text-grid"></fa-icon>
									{{
										$t("footer.savings.modalChartGrid", {
											grid: fmtKw(chargedGrid * 1000, true, false),
										})
									}}
								</div>
							</div>
							<div
								class="chart d-flex justify-content-stretch mb-1 rounded overflow-hidden"
							>
								<div
									v-if="chargedTotal > 0"
									class="chart-item chart-item--self d-flex justify-content-center text-white flex-shrink-1"
									:style="{ width: `${percent}%` }"
								>
									<span class="text-truncate"> {{ percent }}% </span>
								</div>
								<div
									v-if="chargedTotal > 0"
									class="chart-item chart-item--grid d-flex justify-content-center text-white flex-shrink-1"
									:style="{ width: `${100 - percent}%` }"
								>
									<span class="text-truncate"> {{ 100 - percent }}% </span>
								</div>
								<div
									v-if="chargedTotal === 0"
									class="chart-item chart-item--no-data d-flex justify-content-center text-white w-100"
								>
									<span>{{ $t("footer.savings.modalNoData") }}</span>
								</div>
							</div>
						</div>
						<p class="mb-3">
							{{ $t("footer.savings.modalSavingsPrice") }}:
							<strong>{{ pricePerKWh }}</strong>
							<br />
							{{ $t("footer.savings.modalSavingsTotal") }}:
							<strong>{{ savingAmount }}</strong>
						</p>

						<p class="small text-muted mb-3">
							<a
								href="https://github.com/evcc-io/evcc/blob/master/README.md#energy-tariffs--savings-estimate"
								target="_blank"
								class="text-muted"
							>
								{{ $t("footer.savings.modalExplaination") }}</a
							>:
							<span class="text-nowrap">
								{{
									$t("footer.savings.modalExplainationGrid", {
										gridPrice: fmtPricePerKWh(gridPrice, currency),
									})
								}}</span
							>,
							<span class="text-nowrap">
								{{
									$t("footer.savings.modalExplainationFeedIn", {
										feedInPrice: fmtPricePerKWh(feedInPrice, currency),
									})
								}}
							</span>
							<br />
							{{
								$t("footer.savings.modalServerStart", {
									since: fmtTimeAgo(since * -1000),
								})
							}}
						</p>

						<hr class="mb-4" />

						<Sponsor :sponsor="sponsor" class="mb-4" />

						<p class="small text-muted mb-0">
							<strong class="text-primary">
								<fa-icon icon="flask"></fa-icon>
								{{ $t("footer.savings.experimentalLabel") }}:
							</strong>
							{{ $t("footer.savings.experimentalText") }}
							<a
								href="https://github.com/evcc-io/evcc/discussions/2104"
								target="_blank"
								>GitHub Discussions</a
							>.
						</p>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";
import Sponsor from "./Sponsor.vue";

export default {
	name: "Savings",
	components: { Sponsor },
	mixins: [formatter],
	props: {
		selfPercentage: Number,
		since: { type: Number, default: 0 },
		sponsor: String,
		chargedTotal: { type: Number, default: 0 },
		chargedSelfConsumption: { type: Number, default: 0 },
		gridPrice: { type: Number, default: 0.3 },
		feedInPrice: { type: Number, default: 0.08 },
		currency: String,
	},
	computed: {
		chargedGrid() {
			return this.chargedTotal - this.chargedSelfConsumption;
		},
		defaultPrices() {
			const { gridPrice, feedInPrice } = this.$options.propsData;
			return gridPrice === undefined || feedInPrice === undefined;
		},
		savingAmount() {
			const priceDiff = (this.gridPrice - this.feedInPrice) / 100;
			const saving = this.chargedSelfConsumption * priceDiff;
			return this.fmtMoney(saving, this.currency);
		},
		pricePerKWh() {
			const total =
				this.chargedGrid * this.gridPrice + this.chargedSelfConsumption * this.feedInPrice;
			const perKWh = total / this.chargedTotal;
			return this.fmtPricePerKWh(perKWh || this.gridPrice, this.currency);
		},
		percent() {
			return Math.round(this.selfPercentage) || 0;
		},
	},
};
</script>
<style scoped>
/* make modal a bottom drawer on small screens */
@media (max-width: 575px) {
	.modal-dialog.modal-dialog-centered {
		align-items: flex-end;
		margin-bottom: 0;
	}
	.modal.fade .modal-dialog {
		transition: transform 0.4s ease;
		transform: translate(0, 150px);
	}
	.modal.show .modal-dialog {
		transform: none;
	}
	.modal-dialog-scrollable {
		height: calc(100% - 0.5rem);
	}
	.modal-content {
		border-radius: 1rem 1rem 0 0;
	}
}

.chart {
	height: 1.6rem;
}

.chart-item--self {
	background-color: var(--evcc-self);
}
.chart-item--grid {
	background-color: var(--evcc-grid);
}
.chart-item--no-data {
	background-color: var(--bs-gray);
}

.chart-item {
	transition-property: width;
	transition-duration: 500ms;
	transition-timing-function: linear;
}
</style>
