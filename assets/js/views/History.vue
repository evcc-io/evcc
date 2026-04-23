<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader title="History" />
		<div class="alert alert-light mb-5">
			This page is for development purposes only. Helps verify logged data. A proper
			visualization is coming soon, stay tuned.
		</div>
		<div class="row">
			<main class="col-12">
				<div v-if="loading" class="my-5 text-center text-muted">loading...</div>
				<template v-else>
					<section v-if="powerSeries.length" class="mb-5">
						<h3 class="fw-normal mb-3">Power <small class="ms-2">48 hours</small></h3>
						<PowerChart :series="powerSeries" :from="powerFrom" :to="powerTo" />
					</section>
					<section v-if="energySeries.length" class="mb-5">
						<h3 class="fw-normal mb-3">Energy <small class="ms-2">14 days</small></h3>
						<EnergyChart :series="energySeries" :from="energyFrom" :days="14" />
					</section>
					<div
						v-if="!powerSeries.length && !energySeries.length"
						class="my-5 text-center text-muted"
					>
						no data
					</div>
				</template>
			</main>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import Header from "../components/Top/Header.vue";
import PowerChart from "../components/History/PowerChart.vue";
import EnergyChart from "../components/History/EnergyChart.vue";
import api from "../api";

export default defineComponent({
	name: "History",
	components: {
		TopHeader: Header,
		PowerChart,
		EnergyChart,
	},
	data() {
		return {
			powerSeries: [] as any[],
			energySeries: [] as any[],
			powerFrom: new Date(),
			powerTo: new Date(),
			energyFrom: new Date(),
			loading: true,
			interval: null as ReturnType<typeof setInterval> | null,
		};
	},
	head() {
		return { title: "History" };
	},
	mounted() {
		this.fetchData();
		this.interval = setInterval(() => this.fetchData(), 15 * 60 * 1e3);
	},
	unmounted() {
		if (this.interval) {
			clearInterval(this.interval);
		}
	},
	methods: {
		async fetchData() {
			try {
				this.powerTo = new Date();
				this.powerFrom = new Date();
				this.powerFrom.setDate(this.powerFrom.getDate() - 2);
				this.powerFrom.setHours(0, 0, 0, 0);

				this.energyFrom = new Date();
				this.energyFrom.setDate(this.energyFrom.getDate() - 13);
				this.energyFrom.setHours(0, 0, 0, 0);

				const [powerRes, energyRes] = await Promise.all([
					api.get("history/energy", {
						params: {
							from: this.powerFrom.toISOString(),
							to: this.powerTo.toISOString(),
							grouped: true,
						},
					}),
					api.get("history/energy", {
						params: {
							from: this.energyFrom.toISOString(),
							to: this.powerTo.toISOString(),
							aggregate: "day",
							grouped: true,
						},
					}),
				]);

				this.powerSeries = powerRes.data || [];
				this.energySeries = energyRes.data || [];
			} catch (e) {
				console.error("Failed to load energy history", e);
			} finally {
				this.loading = false;
			}
		},
	},
});
</script>
