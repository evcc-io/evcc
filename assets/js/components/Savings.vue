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
		<div id="savingsModal" class="modal fade" tabindex="-1" role="dialog" aria-hidden="true">
			<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							{{ $t("footer.savings.modalTitle", { percent }) }}
						</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<p
							v-html="
								$t('footer.savings.modalText', {
									percent,
									total: fmtKw(chargedTotal, true, false),
									savingEuro,
									since: fmtTimeAgo(since * -1000),
								})
							"
						/>
						<vc-donut
							class="m-4"
							:sections="[
								{
									label: `Eigenstrom: ${fmtKw(
										chargedSelfConsumption,
										true,
										false
									)} kWh`,
									value: this.chargedSelfConsumption,
									color: '#3aba2c',
								},
								{
									label: `Netzstrom: ${fmtKw(
										chargedTotal - chargedSelfConsumption,
										true,
										false
									)} kWh`,
									value: this.chargedTotal - this.chargedSelfConsumption,
									color: '#343a40',
								},
							]"
							:total="chargedTotal"
							has-legend
							:size="110"
							:thickness="100"
							legend-placement="right"
							:auto-adjust-text-size="false"
						></vc-donut>

						<Sponsor :sponsor="sponsor" />

						<p class="small text-muted text-center">
							{{ $t("footer.savings.modalExplaination") }}
							<span class="text-nowrap">
								{{
									$t("footer.savings.modalExplainationGrid", { gridPrice })
								}}</span
							>,
							<span class="text-nowrap">
								{{ $t("footer.savings.modalExplainationFeedin", { feedinPrice }) }}
							</span>
							<a
								href="https://github.com/evcc-io/evcc/blob/master/README.md#energy-tariffs--savings-estimate"
								target="_blank"
								class="text-muted"
								><fa-icon
									v-if="defaultPrices"
									:title="$t('footer.savings.modalExplainationAdjust')"
									icon="wrench"
									class="icon ms-1"
								></fa-icon
								><fa-icon
									v-else
									:title="$t('footer.savings.modalExplainationCalculation')"
									icon="info-circle"
									class="icon ms-1"
								></fa-icon
							></a>
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
	mixins: [formatter],
	components: { Sponsor },
	props: {
		selfPercentage: Number,
		since: Number,
		sponsor: String,
		chargedTotal: Number,
		chargedSelfConsumption: Number,
		gridPrice: { type: Number, default: 30 },
		feedinPrice: { type: Number, default: 8 },
	},
	computed: {
		defaultPrices() {
			const { gridPrice, feedinPrice } = this.$options.propsData;
			return gridPrice === undefined || feedinPrice === undefined;
		},
		savingEuro() {
			const priceDiffEuro = (this.gridPrice - this.feedinPrice) / 100;
			const saving = (this.chargedSelfConsumption / 1000) * priceDiffEuro;
			return this.$n(saving, { style: "currency", currency: "EUR" });
		},
		percent() {
			return Math.round(this.selfPercentage);
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
</style>
