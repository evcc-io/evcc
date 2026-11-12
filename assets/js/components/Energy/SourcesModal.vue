<template>
	<GenericModal id="energySourcesModal" ref="modal" data-testid="energy-sources-modal">
		<template #title>
			{{ title }} <small class="text-muted fw-normal text-nowrap">{{ period }}</small>
		</template>
		<div class="mb-3">
			<table class="table table-borderless text-nowrap mb-0">
				<thead>
					<tr>
						<th
							class="name text-uppercase text-muted small fw-normal ps-0 text-truncate"
						>
							{{ $t("energy.sources.source") }}
						</th>
						<th
							class="text-uppercase text-muted small fw-normal text-end"
							:class="lastIfNoCost"
						>
							{{ $t("energy.sources.energy") }}
						</th>
						<template v-if="hasCost">
							<th
								class="text-uppercase text-muted small fw-normal text-end pe-0 pe-sm-2"
							>
								{{ $t("energy.sources.cost") }}
							</th>
							<th
								class="text-uppercase text-muted small fw-normal text-end pe-0 d-none d-sm-table-cell"
							>
								{{ $t("energy.sources.price") }}
							</th>
						</template>
					</tr>
				</thead>
				<tbody>
					<template v-for="row in rows" :key="row.from">
						<tr :class="{ 'opacity-50': isInactive(row) }">
							<td class="name ps-0 pb-1">
								<div class="d-flex align-items-center gap-2">
									<span
										class="dot rounded-circle flex-shrink-0"
										:style="{ backgroundColor: color(row) }"
									></span>
									<span class="fw-bold tabular">{{
										fmtPercentage(row.share)
									}}</span>
									<span class="text-truncate">{{
										$t(`energy.sources.${row.from}`)
									}}</span>
								</div>
							</td>
							<td class="text-end tabular pb-1" :class="lastIfNoCost">
								{{ fmtEnergy(row.energy) }}
							</td>
							<template v-if="hasCost">
								<td class="text-end tabular pb-1 pe-0 pe-sm-2">
									{{ fmtCost(row) }}
									<!-- phones have no room for the column -->
									<div class="small text-muted d-sm-none">
										{{ fmtPrice(row) }}
									</div>
								</td>
								<td class="text-end tabular pb-1 pe-0 d-none d-sm-table-cell">
									{{ fmtPrice(row) }}
								</td>
							</template>
						</tr>
						<!-- the share as a bar over the full width -->
						<tr aria-hidden="true" :class="{ 'opacity-50': isInactive(row) }">
							<td colspan="4" class="px-0 pt-0 pb-3">
								<div class="track rounded-pill">
									<div
										class="fill rounded-pill"
										:style="{
											width: `${row.share}%`,
											backgroundColor: color(row),
										}"
									></div>
								</div>
							</td>
						</tr>
					</template>
				</tbody>
				<tfoot>
					<tr class="fw-bold align-top">
						<td class="ps-0">{{ $t("energy.sources.total") }}</td>
						<td class="text-end tabular" :class="lastIfNoCost">
							{{ fmtEnergy(totalEnergy) }}
						</td>
						<template v-if="hasCost">
							<td class="text-end tabular pe-0 pe-sm-2">
								{{ fmtMoneyWithSymbol(totalCost, currency) }}
								<div class="small text-muted fw-normal d-sm-none">
									{{ averagePrice }}
								</div>
							</td>
							<td
								class="text-end tabular text-muted fw-normal pe-0 d-none d-sm-table-cell"
							>
								{{ averagePrice }}
							</td>
						</template>
					</tr>
				</tfoot>
			</table>
		</div>
		<p v-if="hasCost" class="small text-muted mb-0">{{ $t("energy.sources.hint") }}</p>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import GenericModal from "../Helper/GenericModal.vue";
import { CURRENCY } from "@/types/evcc";
import type { SourceRow } from "./mix";
import { groupColor } from "./groups";

// where an entity's energy came from, with energy and cost per source
export default defineComponent({
	name: "SourcesModal",
	components: { GenericModal },
	mixins: [formatter],
	props: {
		title: { type: String, default: "" },
		period: { type: String, default: "" }, // the day, month or year the numbers cover
		rows: { type: Array as PropType<SourceRow[]>, default: () => [] },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
	},
	computed: {
		hasCost(): boolean {
			return this.rows.some((row) => row.cost !== undefined);
		},
		// the energy column ends the table without price data
		lastIfNoCost(): string {
			return this.hasCost ? "" : "pe-0";
		},
		// what a kWh cost on average: weighted by energy, over the sources that have a price
		averagePrice(): string {
			const priced = this.rows
				.filter((row) => row.cost !== undefined)
				.reduce((acc, row) => acc + row.energy, 0);
			if (!priced) return "";
			return `ø ${this.fmtPricePerKWh(this.totalCost / priced, this.currency, true)}`;
		},
		totalEnergy(): number {
			return this.rows.reduce((acc, row) => acc + row.energy, 0);
		},
		totalCost(): number {
			return this.rows.reduce((acc, row) => acc + (row.cost ?? 0), 0);
		},
	},
	methods: {
		color(row: SourceRow): string {
			return groupColor(row.from);
		},
		// delivered nothing worth showing, less than half a watt hour
		isInactive(row: SourceRow): boolean {
			return Math.round(row.energy * 1000) === 0;
		},
		// a source that delivered nothing has no price
		fmtPrice(row: SourceRow): string {
			const price = this.isInactive(row) ? 0 : (row.price ?? 0);
			return this.fmtPricePerKWh(price, this.currency, true);
		},
		// one unit for the column, set by its largest row: the rows add up at a glance
		fmtEnergy(kWh: number): string {
			const max = Math.max(0, ...this.rows.map((row) => row.energy));
			return this.fmtWh(kWh * 1000, this.getPowerUnit(max * 1000));
		},
		fmtCost(row: SourceRow): string {
			return this.fmtMoneyWithSymbol(row.cost ?? 0, this.currency);
		},
		open() {
			(this.$refs["modal"] as unknown as { open: () => void }).open();
		},
	},
});
</script>

<style scoped>
.dot {
	width: 0.75rem;
	height: 0.75rem;
}
/* takes the room the numbers leave and gives way first: on a very narrow screen the
   name is cut, the table never scrolls */
.name {
	width: 100%;
	max-width: 0;
}
.track {
	height: 0.5rem;
	background: var(--evcc-background);
}
/* a source that delivered next to nothing still shows as a sliver */
.fill {
	height: 100%;
	min-width: 0.25rem;
	transition: width var(--evcc-transition-medium) ease;
}
</style>
