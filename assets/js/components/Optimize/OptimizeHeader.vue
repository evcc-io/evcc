<template>
	<div class="optimize-header">
		<!-- Summary fields -->
		<div class="fields row g-0">
			<!-- Primary goal -->
			<div class="field col-12 col-md-6 col-lg-3">
				<div class="field-head">
					<div class="field-label small text-uppercase fw-bold evcc-gray">
						Primary goal
					</div>
					<div class="field-caption small evcc-gray mt-1">always, fixed</div>
				</div>
				<div class="field-value gap-2">
					<span class="large fw-bold text-lowercase evcc-default-text">Lowest cost</span>
					<LockIcon class="value-icon" />
				</div>
			</div>

			<!-- Secondary goal (only interactive setting) -->
			<div class="field col-12 col-md-6 col-lg-3">
				<div class="field-head">
					<div class="field-label small text-uppercase fw-bold evcc-gray">
						Secondary goal
					</div>
					<div class="field-caption small evcc-gray mt-1">only on a tie</div>
				</div>
				<div class="field-value gap-2">
					<CustomSelect
						id="optimizerSecondaryGoal"
						:options="strategyOptions"
						:selected="selectedStrategy"
						@change="onStrategyChange"
					>
						<span
							class="large fw-bold text-lowercase evcc-default-text secondary-goal"
							>{{ secondaryGoalLabel }}</span
						>
					</CustomSelect>
				</div>
			</div>

			<!-- Result -->
			<div class="field col-12 col-md-6 col-lg-3">
				<div class="field-head">
					<div class="field-label small text-uppercase fw-bold evcc-gray">Result</div>
					<div class="field-caption small evcc-gray mt-1">
						<span v-if="relativeTime" class="tabular">{{ relativeTime }}, </span>
						<span v-if="pending" class="updating fw-bold text-decoration-underline"
							>updating…</span
						>
						<button
							v-else
							type="button"
							class="btn btn-link p-0 align-baseline fw-bold text-decoration-underline update-now"
							@click="$emit('optimize')"
						>
							update now
						</button>
					</div>
				</div>
				<div class="field-value gap-2">
					<StatusIndicator :variant="statusVariant">
						<span class="large fw-bold text-lowercase evcc-default-text">{{
							status
						}}</span>
					</StatusIndicator>
					<span
						ref="statusInfo"
						class="value-icon info-icon"
						data-bs-toggle="tooltip"
						role="button"
						tabindex="0"
					>
						<shopicon-regular-info></shopicon-regular-info>
					</span>
				</div>
			</div>

			<!-- Net grid cost -->
			<div class="field col-12 col-md-6 col-lg-3">
				<div class="field-head">
					<div class="field-label small text-uppercase fw-bold evcc-gray">
						Net grid cost
					</div>
					<div class="field-caption small evcc-gray mt-1">
						over the next {{ horizonHours }} hours
					</div>
				</div>
				<div class="field-value gap-2">
					<span
						class="fs-4 fw-bold tabular"
						:class="isCredit ? 'text-primary' : 'evcc-default-text'"
					>
						{{ netCostDisplay }}
					</span>
					<span
						ref="costInfo"
						class="value-icon info-icon"
						data-bs-toggle="tooltip"
						role="button"
						tabindex="0"
					>
						<shopicon-regular-info></shopicon-regular-info>
					</span>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Tooltip from "bootstrap/js/dist/tooltip";
import CustomSelect from "../Helper/CustomSelect.vue";
import StatusIndicator from "../Config/StatusIndicator.vue";
import LockIcon from "../MaterialIcon/Lock.vue";
import formatter from "@/mixins/formatter";
import minuteTicker from "@/mixins/minuteTicker";
import { CURRENCY, OptimizationStatus } from "@/types/evcc";
import "@h2d2/shopicons/es/regular/info";

// human labels for the optimizer grid charging strategies; the available values
// themselves come from backend state to avoid drift if the enum changes
const STRATEGY_LABELS: Record<string, string> = {
	charge_before_export: "fill battery first",
	attenuate_grid_peaks: "reduce grid peaks",
	none: "no preference",
};

const STATUS_TOOLTIP =
	"The optimizer result:<br><br>" +
	"<strong>Optimal</strong>: the best plan was found.<br>" +
	"<strong>Infeasible</strong>: the constraints cannot all be met.<br>" +
	"<strong>Not solved</strong>: optimization hasn't finished yet.";

const COST_TOOLTIP =
	"Forecasted net grid cost over the optimization window: energy bought minus sold, " +
	"with leftover battery charge credited at the lowest price. Lower is better, " +
	"positive means you pay the grid, negative means you receive a credit.";

export default defineComponent({
	name: "OptimizeHeader",
	components: {
		CustomSelect,
		StatusIndicator,
		LockIcon,
	},
	mixins: [formatter, minuteTicker],
	props: {
		updated: { type: String, default: "" },
		status: { type: String as PropType<OptimizationStatus>, default: undefined },
		netCost: { type: Number, default: 0 },
		horizonHours: { type: Number, default: 0 },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		chargingStrategies: { type: Array as PropType<string[]>, default: () => [] },
		selectedStrategy: { type: String, default: "" },
		pending: { type: Boolean, default: false },
	},
	emits: ["optimize", "change-strategy"],
	data() {
		return {
			tooltips: [] as Tooltip[],
		};
	},
	computed: {
		strategyOptions() {
			return this.chargingStrategies.map((value) => ({
				value,
				name: STRATEGY_LABELS[value] || value,
			}));
		},
		secondaryGoalLabel(): string {
			return STRATEGY_LABELS[this.selectedStrategy] || this.selectedStrategy;
		},
		relativeTime(): string {
			if (!this.updated) return "";
			const elapsed = new Date(this.updated).getTime() - this.everyMinute.getTime();
			if (Math.abs(elapsed) < 60 * 1000) return "just now";
			return this.fmtTimeAgo(elapsed);
		},
		statusVariant(): "success" | "warning" | "muted" {
			switch (this.status) {
				case OptimizationStatus.OPTIMAL:
					return "success";
				case OptimizationStatus.INFEASIBLE:
				case OptimizationStatus.UNBOUNDED:
					return "warning";
				default:
					return "muted";
			}
		},
		isCredit(): boolean {
			return this.netCost < 0;
		},
		netCostDisplay(): string {
			return this.fmtMoney(this.netCost, this.currency, true, true);
		},
	},
	mounted() {
		this.initTooltips();
	},
	beforeUnmount() {
		this.tooltips.forEach((t) => t.dispose());
	},
	methods: {
		onStrategyChange(e: Event) {
			this.$emit("change-strategy", (e.target as HTMLSelectElement).value);
		},
		initTooltips() {
			const items: [Element | undefined, string][] = [
				[this.$refs["statusInfo"] as Element | undefined, STATUS_TOOLTIP],
				[this.$refs["costInfo"] as Element | undefined, COST_TOOLTIP],
			];
			for (const [el, title] of items) {
				if (el) {
					// left-align via inner markup: tooltips render at <body>, outside scoped styles
					this.tooltips.push(
						new Tooltip(el, {
							title: `<div class="text-start">${title}</div>`,
							html: true,
						})
					);
				}
			}
		},
	},
});
</script>

<style scoped>
/* underline + weight come from utility classes; only the gray/hover color is custom */
.update-now {
	font-size: inherit;
	color: var(--evcc-gray);
}
.update-now:hover {
	color: var(--bs-primary);
}
.updating {
	font-size: inherit;
	color: var(--evcc-gray);
	cursor: progress;
}

/* small: each field a full-width row (Bootstrap col-12), label left, value right */
.field {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 1rem;
	padding: 0.75rem 0;
	border-top: 1px solid var(--evcc-gray-25);
}
.field:last-child {
	border-bottom: 1px solid var(--evcc-gray-25);
}
.field-head {
	display: flex;
	flex-direction: column;
}
.field-value {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	text-align: right;
}
.value-icon {
	display: inline-flex;
	align-items: center;
	flex-shrink: 0;
	color: var(--evcc-gray);
}
.info-icon {
	cursor: help;
}
.secondary-goal {
	text-decoration: underline;
	text-decoration-color: var(--evcc-gray);
	text-underline-offset: 3px;
}

/* md and up: each field a vertical stack (label, value, caption); columns via Bootstrap grid */
@media (min-width: 768px) {
	.field {
		flex-direction: column;
		justify-content: flex-start;
		align-items: flex-start;
		gap: 0;
		padding: 0.75rem 1.5rem;
		border: none;
	}
	.field:last-child {
		border-bottom: none;
	}
	/* promote label + caption so value can sit between them via order */
	.field-head {
		display: contents;
	}
	.field-label {
		order: 0;
		margin-bottom: 0.5rem;
	}
	.field-value {
		order: 1;
		justify-content: flex-start;
		text-align: left;
	}
	.field-caption {
		order: 2;
	}
}

/* medium (col-md-6): two columns, dividers between */
@media (min-width: 768px) and (max-width: 991.98px) {
	.field:nth-child(odd) {
		padding-left: 0;
	}
	.field:nth-child(even) {
		padding-right: 0;
		border-left: 1px solid var(--evcc-gray-25);
	}
	.field:nth-child(n + 3) {
		border-top: 1px solid var(--evcc-gray-25);
	}
}

/* large (col-lg-3): four equal blocks in one row, dividers between */
@media (min-width: 992px) {
	.field {
		padding-block: 0;
		border-left: 1px solid var(--evcc-gray-25);
	}
	.field:first-child {
		padding-left: 0;
		border-left: none;
	}
	.field:last-child {
		padding-right: 0;
	}
}
</style>
