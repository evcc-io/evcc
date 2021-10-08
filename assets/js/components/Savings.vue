<template>
	<div v-if="selfPercentage > 1">
		<div class="button-wrap position-absolute top-50 start-50 translate-middle">
			<button
				class="
					px-2
					py-1
					d-flex d-lg-none
					bg-primary
					rounded-pill
					border-0
					align-items-center
					text-white
				"
				data-bs-toggle="modal"
				data-bs-target="#savingsModal"
			>
				<fa-icon icon="sun" class="icon me-1"></fa-icon>
				<span> {{ $t("footer.savings.footerShort", { percent }) }}</span>
			</button>
			<button
				class="d-none d-lg-flex px-2 py-1 bg-white rounded-pill border-0 align-items-center"
				data-bs-toggle="modal"
				data-bs-target="#savingsModal"
			>
				<fa-icon icon="sun" class="icon me-2 text-evcc"></fa-icon>
				<span class="text-dark">{{ $t("footer.savings.footerLong", { percent }) }}</span>
			</button>
		</div>
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
									since: fmtTimeAgo(since * 1000),
									total: fmtKw(chargedTotal, true, false),
									self: fmtKw(chargedSelfConsumption, true, false),
									savingEuro,
								})
							"
						/>
						<div v-if="sponsor" class="d-flex justify-content-center my-4">
							<button class="btn btn-outline-success">
								<fa-icon :icon="['far', 'heart']" class="icon me-1"></fa-icon>
								Weiterentwicklung untersützen
							</button>
						</div>
						<div class="d-flex flex-column align-items-center my-4">
							<button class="btn btn-success mb-2">
								<fa-icon :icon="['fas', 'heart']" class="icon me-1"></fa-icon>
								Danke naltatis! Lust auf Konfetti?
							</button>
							<a href="#" class="small text-muted text-decoration-none"
								>...oder evcc Sticker?</a
							>
						</div>
						<p>
							<small class="text-muted">{{
								$t("footer.savings.modalExplaination", {
									gridPrice: gridPrice * 100,
									feedinPrice: feedinPrice * 100,
								})
							}}</small>
						</p>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";

export default {
	name: "Savings",
	mixins: [formatter],
	props: {
		selfPercentage: Number,
		since: Number,
		sponsor: String,
		chargedTotal: Number,
		chargedSelfConsumption: Number,
		gridPrice: { type: Number, default: 0.3 },
		feedinPrice: { type: Number, default: 0.08 },
	},
	computed: {
		savingEuro() {
			const priceDiff = this.gridPrice - this.feedinPrice;
			const saving = (this.chargedSelfConsumption / 1000) * priceDiff;
			return this.$n(saving, { style: "currency", currency: "EUR" });
		},
		percent() {
			return Math.round(this.selfPercentage);
		},
	},
	methods: {},
};
</script>
