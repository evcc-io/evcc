<template>
	<div class="collapsible-wrapper" :class="{ open: visible }">
		<div class="collapsible-content ring-space">
			<div
				class="round-box p-4 pb-3 my-4 border-primary border-opacity-50"
				data-testid="setup-checklist"
			>
				<div class="d-flex justify-content-between align-items-baseline gap-3 mb-2">
					<h5 class="mb-0">{{ $t("onboarding.checklist.title") }}</h5>
					<span class="small evcc-gray text-nowrap">
						{{ $t("onboarding.checklist.suggestions", openRows.length) }}
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
							:class="{
								'evcc-gray text-decoration-line-through': row.state === 'dismissed',
							}"
						>
							{{ $t(`onboarding.item.${row.item}`) }}
						</span>
						<span
							v-if="row.state === 'configured'"
							class="d-flex align-items-center gap-1 small evcc-gray flex-shrink-0"
						>
							<CheckIcon class="text-primary" />
							{{ $t("config.general.configured") }}
						</span>
						<!-- disabled buttons get no hover events, the wrapper carries the hint -->
						<span
							v-if="row.state === 'open'"
							v-tooltip="
								row.blocked ? $t('onboarding.checklist.legacyTariffs') : undefined
							"
						>
							<button
								type="button"
								class="btn btn-sm btn-outline-primary btn-pill px-3"
								:disabled="row.blocked"
								@click="add(row.item)"
							>
								{{ $t("config.general.add") }}
							</button>
						</span>
						<button
							v-if="row.state !== 'configured'"
							type="button"
							class="btn btn-sm btn-link evcc-gray text-decoration-none p-0 flex-shrink-0"
							:aria-pressed="row.state === 'dismissed'"
							:aria-label="dismissLabel(row)"
							@click="toggle(row)"
						>
							<!-- narrow screens: icon, the add button needs the room -->
							<span :class="{ 'd-none d-sm-inline': row.state === 'open' }">
								{{ dismissLabel(row) }}
							</span>
							<shopicon-regular-close
								v-if="row.state === 'open'"
								size="s"
								class="d-sm-none"
							></shopicon-regular-close>
						</button>
					</li>
				</ul>
				<div class="d-flex justify-content-end pt-2">
					<button
						type="button"
						class="btn btn-sm btn-link evcc-gray text-decoration-none p-0"
						@click="dismissAll"
					>
						{{ $t("config.general.dismissAll") }}
					</button>
				</div>
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
import { dismiss, undismiss, ONBOARDING_ITEMS } from "@/onboarding";
import { ONBOARDING_ICONS } from "./Onboarding/icons";

const METER_ITEMS = ["battery", "pv", "grid", "consumer"];
const TARIFF_TYPES: Partial<Record<OnboardingItem, string>> = {
	gridImport: "grid",
	gridExport: "feedIn",
};
const TARIFF_ITEMS: OnboardingItem[] = ["gridImport", "gridExport", "co2", "solar"];

interface Row {
	item: OnboardingItem;
	state: "configured" | "dismissed" | "open";
	blocked: boolean;
}

// rows stay once done (configured or dismissed), the card collapses when nothing is left
export default defineComponent({
	name: "SetupChecklist",
	components: { CheckIcon },
	props: {
		configured: { type: Object as PropType<Record<string, boolean>>, required: true },
		dismissed: { type: Array as PropType<OnboardingItem[]>, default: () => [] },
		legacyTariffs: Boolean,
	},
	data() {
		return { ICONS: ONBOARDING_ICONS };
	},
	computed: {
		rows(): Row[] {
			return ONBOARDING_ITEMS.map((item) => {
				let state: Row["state"] = "open";
				if (this.configured[item]) state = "configured";
				else if (this.dismissed.includes(item)) state = "dismissed";
				return {
					item,
					state,
					blocked: this.legacyTariffs && TARIFF_ITEMS.includes(item),
				};
			});
		},
		openRows(): Row[] {
			return this.rows.filter((r) => r.state === "open");
		},
		visible() {
			return this.openRows.length > 0;
		},
		columnRows() {
			return Math.ceil(this.rows.length / 2);
		},
		progress() {
			return Math.round(((this.rows.length - this.openRows.length) / this.rows.length) * 100);
		},
	},
	methods: {
		dismissLabel(row: Row) {
			return this.$t(`config.general.${row.state === "dismissed" ? "dismissed" : "dismiss"}`);
		},
		toggle(row: Row) {
			return row.state === "dismissed" ? undismiss(row.item) : dismiss(row.item);
		},
		dismissAll() {
			return dismiss(...this.openRows.map((r) => r.item));
		},
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
/* done rows (checkmark) as tall as rows with the add button */
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
