<template>
	<div class="collapsible-wrapper" :class="{ open: visible }">
		<div class="collapsible-content ring-space">
			<div
				class="round-box p-4 my-4 border-primary border-opacity-50"
				data-testid="setup-checklist"
			>
				<div class="d-flex justify-content-between align-items-baseline gap-3 mb-2">
					<h5 class="mb-0">{{ $t("onboarding.checklist.title") }}</h5>
					<span class="small evcc-gray text-nowrap">
						{{
							$t("onboarding.checklist.progress", {
								done: doneCount,
								total: rows.length,
							})
						}}
					</span>
				</div>
				<div class="progress mb-3" role="progressbar" :aria-valuenow="progress">
					<div class="progress-bar bg-primary" :style="{ width: `${progress}%` }"></div>
				</div>
				<ul class="list-unstyled mb-0 items" :style="{ '--rows': columnRows }">
					<li
						v-for="(row, i) in rows"
						:key="row.item"
						class="d-flex align-items-center gap-3 py-2 border-bottom min-w-0"
						:class="{ 'column-end': i === columnRows - 1 }"
						:data-testid="`checklist-${row.item}`"
					>
						<component
							:is="ICONS[row.item]"
							size="s"
							class="icon-row evcc-gray flex-shrink-0"
						/>
						<span
							class="flex-grow-1 text-truncate"
							:class="{ 'evcc-gray text-decoration-line-through': row.done }"
						>
							{{ $t(`onboarding.item.${row.item}`) }}
						</span>
						<CheckIcon v-if="row.done" class="text-primary flex-shrink-0" />
						<template v-else>
							<button
								type="button"
								class="btn btn-sm btn-outline-primary btn-pill px-3"
								@click="add(row.item)"
							>
								{{ $t("config.general.add") }}
							</button>
							<button
								type="button"
								class="btn btn-sm btn-link evcc-gray p-0"
								:aria-label="$t('onboarding.checklist.dismiss')"
								:title="$t('onboarding.checklist.dismiss')"
								@click="dismiss(row.item)"
							>
								<shopicon-regular-close size="s"></shopicon-regular-close>
							</button>
						</template>
					</li>
				</ul>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/close";
import CheckIcon from "../MaterialIcon/Check.vue";
import { defineComponent, type PropType } from "vue";
import type { OnboardingItem } from "@/types/evcc";
import { openModal } from "@/configModal";
import { dismiss, ONBOARDING_ITEMS } from "@/onboarding";
import { ONBOARDING_ICONS } from "./Onboarding/icons";

const METER_ITEMS = ["battery", "pv", "grid", "consumer"];
const TARIFF_TYPES: Partial<Record<OnboardingItem, string>> = {
	gridImport: "grid",
	gridExport: "feedIn",
};

// rows stay once done (configured or dismissed), the card collapses when nothing is left
export default defineComponent({
	name: "SetupChecklist",
	components: { CheckIcon },
	props: {
		configured: { type: Object as PropType<Record<string, boolean>>, required: true },
		dismissed: { type: Array as PropType<OnboardingItem[]>, default: () => [] },
	},
	data() {
		return { ICONS: ONBOARDING_ICONS };
	},
	computed: {
		rows() {
			// charging point and heater complement each other
			const control = !!(this.configured["loadpoint"] || this.configured["heater"]);
			return ONBOARDING_ITEMS.map((item) => ({
				item,
				done:
					!!this.configured[item] ||
					this.dismissed.includes(item) ||
					((item === "loadpoint" || item === "heater") && control),
			}));
		},
		visible() {
			return this.rows.some((r) => !r.done);
		},
		columnRows() {
			return Math.ceil(this.rows.length / 2);
		},
		doneCount() {
			return this.rows.filter((r) => r.done).length;
		},
		progress() {
			return Math.round((this.doneCount / this.rows.length) * 100);
		},
	},
	methods: {
		dismiss,
		add(item: OnboardingItem) {
			if (METER_ITEMS.includes(item)) {
				openModal("meter", { type: item });
			} else if (item === "loadpoint" || item === "heater") {
				openModal("loadpoint", { type: item === "heater" ? "heating" : "charging" });
			} else {
				openModal("tariff", { type: TARIFF_TYPES[item] ?? item });
			}
		},
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

.progress {
	--bs-progress-height: 4px;
	--bs-progress-bg: var(--evcc-gray-25);
}
/* done rows (checkmark) as tall as rows with the add button, last row padding balances the card padding */
.items {
	margin-bottom: -0.5rem;
}
.items li {
	min-height: 3rem;
}
.items li:last-child {
	border-bottom: 0 !important;
}
/* fill the first column top to bottom, then the second */
@media (--md-and-up) {
	.items {
		display: grid;
		grid-auto-flow: column;
		grid-template-rows: repeat(var(--rows), auto);
		grid-template-columns: repeat(2, minmax(0, 1fr));
		column-gap: 2rem;
	}
	.items .column-end {
		border-bottom: 0 !important;
	}
}
</style>
