<template>
	<GenericModal id="energySourcesModal" ref="modal" data-testid="energy-sources-modal">
		<template #title>
			{{ title }} <small class="text-muted fw-normal text-nowrap">{{ period }}</small>
		</template>
		<div class="table-responsive mb-3">
			<table class="table table-borderless text-nowrap mb-0">
				<thead>
					<tr>
						<th class="text-uppercase text-muted small fw-normal ps-0">
							{{ $t("energy.sources.source") }}
						</th>
						<th class="text-uppercase text-muted small fw-normal text-end">
							{{ $t("energy.sources.share") }}
						</th>
						<th
							class="text-uppercase text-muted small fw-normal text-end"
							:class="lastIfNoCost"
						>
							{{ $t("energy.sources.energy") }}
						</th>
						<template v-if="hasCost">
							<th class="text-uppercase text-muted small fw-normal text-end">
								{{ $t("energy.sources.cost") }}
							</th>
							<th class="text-uppercase text-muted small fw-normal text-end pe-0">
								{{ $t("energy.sources.price") }}
							</th>
						</template>
					</tr>
				</thead>
				<tbody>
					<tr
						v-for="row in rows"
						:key="row.from"
						:class="{ 'opacity-50': isInactive(row) }"
					>
						<td class="ps-0">{{ $t(`energy.sources.${row.from}`) }}</td>
						<td class="text-end">{{ fmtPercentage(row.share) }}</td>
						<td class="text-end" :class="lastIfNoCost">
							{{ fmtKWh(row.energy) }}
						</td>
						<template v-if="hasCost">
							<td class="text-end">{{ fmtCost(row) }}</td>
							<td class="text-end pe-0">{{ fmtPrice(row) }}</td>
						</template>
					</tr>
				</tbody>
				<tfoot>
					<tr class="fw-bold border-top">
						<td class="ps-0">{{ $t("energy.sources.total") }}</td>
						<td></td>
						<td class="text-end" :class="lastIfNoCost">
							{{ fmtKWh(totalEnergy) }}
						</td>
						<template v-if="hasCost">
							<td class="text-end">{{ fmtMoneyWithSymbol(totalCost, currency) }}</td>
							<td></td>
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
		totalEnergy(): number {
			return this.rows.reduce((acc, row) => acc + row.energy, 0);
		},
		totalCost(): number {
			return this.rows.reduce((acc, row) => acc + (row.cost ?? 0), 0);
		},
	},
	methods: {
		// delivered nothing worth showing, less than half a watt hour
		isInactive(row: SourceRow): boolean {
			return Math.round(row.energy * 1000) === 0;
		},
		// a source that delivered nothing has no price
		fmtPrice(row: SourceRow): string {
			const price = this.isInactive(row) ? 0 : (row.price ?? 0);
			return this.fmtPricePerKWh(price, this.currency, true);
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
