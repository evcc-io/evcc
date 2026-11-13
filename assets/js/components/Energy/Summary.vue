<template>
	<Card edge-to-edge class="box-pull-out mb-4" data-testid="energy-summary" data-anchor>
		<div class="row gx-4 gx-xl-5 gy-4">
			<div
				v-for="ring in rings"
				:key="ring.key"
				class="col-12 col-md-6 col-xl-4"
				:data-testid="`energy-ring-${ring.key}`"
			>
				<div class="label text-uppercase mb-3">{{ ring.label }}</div>
				<div class="d-flex align-items-center gap-3">
					<ShareRing :segments="ring.segments" :size="120" :stroke="3">
						<div class="fs-5 fw-bold lh-1 text-nowrap tabular">{{ ring.value }}</div>
						<div class="label mt-1">{{ ring.unit }}</div>
					</ShareRing>
					<ul class="list-unstyled m-0 d-flex flex-column gap-2 overflow-hidden">
						<li
							v-for="part in ring.parts"
							:key="part.label"
							class="d-flex align-items-baseline gap-2"
						>
							<span
								class="dot rounded-circle flex-shrink-0 align-self-center"
								:style="{ backgroundColor: part.color }"
							></span>
							<span class="share fw-bold text-end tabular">{{ part.share }}</span>
							<span class="text-truncate">{{ part.label }}</span>
						</li>
					</ul>
				</div>
			</div>
			<div class="col-12 col-xl-4">
				<StatRow :stats="stats" item-class="col-6 col-md-3 col-xl-6" />
			</div>
		</div>
	</Card>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Card from "../Helper/Card.vue";
import StatRow from "./StatRow.vue";
import ShareRing from "./ShareRing.vue";
import formatter from "@/mixins/formatter";
import colors from "@/colors";
import { CURRENCY } from "@/types/evcc";
import type { Flow, FlowCo2, FlowCost, StatItem } from "./types";
import { sinkCost } from "./mix";
import { groupColor } from "./groups";

interface Ring {
	key: string;
	label: string;
	value: string;
	unit: string;
	segments: { value: number; color: string }[];
	// the listed segments with their share
	parts: { label: string; color: string; share: string }[];
}

// the period at a glance: where the production went, where the consumption came
// from, and what own energy saved
export default defineComponent({
	name: "Summary",
	components: { Card, StatRow, ShareRing },
	mixins: [formatter],
	props: {
		autarky: { type: Number, default: 0 }, // percent
		production: { type: Number, default: 0 }, // kWh, metered
		consumption: { type: Number, default: 0 }, // kWh, metered
		gridImport: { type: Number, default: 0 }, // kWh, metered
		// a battery was active in the period, its shares show even at zero
		hasBattery: Boolean,
		// tariffs that are set up now. A period without their data then only says so,
		// there is nothing left to configure
		hasPriceTariff: Boolean,
		hasCo2Tariff: Boolean,
		flows: { type: Array as PropType<Flow[]>, default: () => [] },
		cost: { type: Object as PropType<FlowCost> },
		co2: { type: Object as PropType<FlowCo2> },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
	},
	computed: {
		rings(): Ring[] {
			const used = (f: Flow) => f.to === "home" || f.to === "loadpoint";
			const production = this.ring("production", this.$t("energy.flow.pv"), this.production, [
				{
					label: this.$t("main.energyflow.selfConsumption"),
					color: groupColor("pv"),
					value: this.sum((f) => f.from === "pv" && used(f)),
				},
				{
					label: this.$t("energy.flow.batteryCharge"),
					color: groupColor("battery"),
					value: this.sum((f) => f.from === "pv" && f.to === "battery"),
					hidden: !this.hasBattery,
				},
				{
					label: this.$t("energy.flow.export"),
					color: colors.export || "",
					value: this.sum((f) => f.from === "pv" && f.to === "export"),
				},
			]);
			const consumption = this.ring(
				"consumption",
				this.$t("energy.flow.home"),
				this.consumption,
				(["pv", "battery", "grid"] as const).map((from) => ({
					label: this.$t(`energy.sources.${from}`),
					color: groupColor(from),
					value: this.sum((f) => f.from === from && used(f)),
					hidden: from === "battery" && !this.hasBattery,
				}))
			);
			return [production, consumption].filter((r) => r.segments.some((s) => s.value > 0));
		},
		stats(): StatItem[] {
			const list: StatItem[] = [
				{
					key: "autarky",
					accent: "text-accent2",
					label: this.$t("energy.stat.autarky"),
					number: this.autarky,
					format: (v: number) => this.fmtPercentage(v),
					sub: this.$t("energy.stat.fromGrid", { value: this.fmtKWh(this.gridImport) }),
				},
			];
			if (this.cost) {
				// consumption priced grid only minus what it cost with own energy at the feed-in price
				const { cost, energy } = sinkCost(this.flows, "home", "loadpoint");
				const savings = this.cost.avgGrid * energy - cost;
				const effective = energy ? cost / energy : 0;
				const fmt = (v: number) => this.fmtPricePerKWh(v, this.currency, false, false);
				list.push({
					key: "savings",
					accent: "text-accent2",
					label: this.$t("energy.stat.savings"),
					number: savings,
					format: (v: number) => this.fmtAmount(v),
					// effective price per kWh consumed vs. the average grid price
					sub: this.$t("energy.stat.versus", {
						value: fmt(effective),
						reference: this.fmtPricePerKWh(this.cost.avgGrid, this.currency),
					}),
					tooltip: this.$t("energy.stat.savingsTooltip", {
						energy: this.fmtKWh(energy),
					}),
				});
			} else {
				list.push({
					key: "savings",
					label: this.$t("energy.stat.savings"),
					empty: this.$t("energy.stat.noPrice"),
					configure: !this.hasPriceTariff,
				});
			}
			if (this.cost) {
				const { import: money, importEnergy } = this.cost;
				list.push({
					key: "gridCost",
					label: this.$t("energy.grid.cost"),
					number: money,
					format: (v: number) => this.fmtAmount(v),
					sub: importEnergy
						? `ø ${this.fmtPricePerKWh(money / importEnergy, this.currency)}`
						: "",
				});
			}
			if (this.co2) {
				const saved = this.co2.baseline - this.co2.consumption;
				const consumption = this.co2.consumptionEnergy;
				const effective = consumption ? (this.co2.consumption * 1000) / consumption : 0;
				list.push({
					key: "co2",
					accent: "text-accent3",
					label: this.$t("energy.stat.co2Saved"),
					number: saved * 1000,
					format: (v: number) => this.fmtGrams(v),
					sub: this.$t("energy.stat.versus", {
						value: this.fmtNumber(effective, 0),
						reference: this.fmtCo2Medium(this.co2.avgCo2),
					}),
					tooltip: this.$t("energy.stat.co2SavedTooltip", {
						energy: this.fmtKWh(this.co2.consumptionEnergy),
					}),
				});
			} else {
				list.push({
					key: "co2",
					label: this.$t("energy.stat.co2Saved"),
					empty: this.$t("energy.stat.noCo2"),
					configure: !this.hasCo2Tariff,
				});
			}
			return list;
		},
	},
	methods: {
		// from a thousand on the cents go, the tile has no room for them and they say nothing
		fmtAmount(v: number): string {
			const symbol = this.fmtCurrencySymbol(this.currency);
			return `${this.fmtMoney(v, this.currency, Math.abs(v) < 1000)} ${symbol}`;
		},
		sum(match: (f: Flow) => boolean): number {
			return this.flows.filter(match).reduce((acc, f) => acc + f.energy, 0);
		},
		// every part with its share, also at zero: the list keeps its rows between periods.
		// Only a battery the site does not have is left out
		ring(
			key: string,
			label: string,
			total: number,
			parts: { label: string; color: string; value: number; hidden?: boolean }[]
		): Ring {
			const live = parts.filter((p) => !p.hidden);
			const sum = live.reduce((acc, p) => acc + p.value, 0);
			// the unit goes on its own line below the number
			const [value = "", unit = ""] = this.fmtKWh(total).split(" ");
			return {
				key,
				label,
				value,
				unit,
				segments: parts,
				parts: live.map((p) => ({
					label: p.label,
					color: p.color,
					share: this.fmtPercentage(sum ? (p.value / sum) * 100 : 0),
				})),
			};
		},
	},
});
</script>

<style scoped>
.label {
	color: var(--evcc-gray);
	font-size: 14px;
}
.dot {
	width: 0.75rem;
	height: 0.75rem;
}
.share {
	min-width: 2.5rem;
}
</style>
