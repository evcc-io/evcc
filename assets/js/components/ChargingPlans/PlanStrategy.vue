<template>
	<div data-testid="plan-strategy" class="mb-5 mb-lg-4">
		<div :class="{ 'mb-2': open }">
			<button
				type="button"
				class="btn btn-link btn-sm text-gray p-0 border-0 d-flex align-items-center text-start"
				:class="{ 'text-primary': open }"
				@click="open = !open"
			>
				<span class="fw-bold">
					{{ $t("main.chargingPlan.strategy.label") }}<span v-if="!open">:</span>
				</span>
				<span v-if="!open" class="ms-1">{{ summary }}</span>
				<DropdownIcon class="icon flex-shrink-0" :class="{ iconUp: open }" />
			</button>
		</div>
		<div class="collapsible-wrapper" :class="{ open }">
			<div class="collapsible-content ring-space">
				<div class="row">
					<div class="col-12 col-lg-6 mb-3">
						<div class="row">
							<label
								:for="formId('continuous')"
								class="col-form-label col-5 col-lg-12"
							>
								{{ $t("main.chargingPlan.optimization.label") }}
							</label>
							<div class="col-7 col-lg-12">
								<select
									v-if="disabled"
									:id="formId('continuous')"
									class="form-select"
									disabled
								>
									<option>{{ $t("general.none") }}</option>
								</select>
								<select
									v-else
									:id="formId('continuous')"
									v-model="localContinuous"
									class="form-select"
									@change="updateStrategy"
								>
									<option :value="false">
										{{ $t(`main.chargingPlan.optimization.${cheapestKey}`) }}
									</option>
									<option :value="true">
										{{ $t("main.chargingPlan.optimization.continuous") }}
									</option>
								</select>
							</div>
						</div>
						<div class="small text-muted mt-1">
							{{ optimizationDescription }}
						</div>
					</div>
					<div class="col-12 col-lg-6 mb-3">
						<div class="row">
							<label
								:for="formId('precondition')"
								class="col-form-label col-5 col-lg-12"
							>
								{{ $t("main.chargingPlan.precondition.label") }}
							</label>
							<div class="col-7 col-lg-12">
								<select
									v-if="disabled"
									:id="formId('precondition')"
									class="form-select"
									disabled
								>
									<option>
										{{ $t("main.chargingPlan.precondition.optionAll") }}
									</option>
								</select>
								<select
									v-else
									:id="formId('precondition')"
									v-model="localPrecondition"
									class="form-select"
									@change="updateStrategy"
								>
									<option :value="0">
										{{ $t("main.chargingPlan.precondition.optionNo") }}
									</option>
									<option
										v-for="opt in preconditionOptions"
										:key="opt.value"
										:value="opt.value"
									>
										{{ opt.name }}
									</option>
								</select>
							</div>
						</div>
						<div class="small text-muted mt-1">
							{{
								$t(
									`main.chargingPlan.precondition.${disabled ? "disabledDescription" : "description"}`
								)
							}}
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import { SMART_COST_TYPE, type PlanStrategy } from "@/types/evcc";
import DropdownIcon from "../MaterialIcon/Dropdown.vue";

const HOUR = 60 * 60;
const QUARTER_HOUR = 0.25 * HOUR;
const HALF_HOUR = 0.5 * HOUR;
const ONE_HOUR = 1 * HOUR;
const TWO_HOURS = 2 * HOUR;
const EVERYTHING = 7 * 24 * HOUR;

export default defineComponent({
	name: "ChargingPlanStrategy",
	components: { DropdownIcon },
	mixins: [formatter],
	props: {
		id: [String, Number],
		precondition: { type: Number, default: 0 },
		continuous: { type: Boolean, default: false },
		smartCostType: String as PropType<SMART_COST_TYPE>,
	},
	emits: ["update"],
	data() {
		return {
			open: false,
			localPrecondition: this.precondition,
			localContinuous: this.continuous,
		};
	},
	computed: {
		disabled(): boolean {
			// options only make sense with a dynamic planner tariff
			return !this.smartCostType || this.smartCostType === SMART_COST_TYPE.PRICE_STATIC;
		},
		isCo2(): boolean {
			return this.smartCostType === SMART_COST_TYPE.CO2;
		},
		cheapestKey(): string {
			return this.isCo2 ? "cleanest" : "cheapest";
		},
		summary(): string {
			if (this.disabled) {
				return this.$t("main.chargingPlan.precondition.summaryAll");
			}
			const parts = [
				this.$t(
					`main.chargingPlan.optimization.${this.continuous ? "continuous" : this.cheapestKey}`
				),
			];
			if (this.precondition >= EVERYTHING) {
				parts.push(this.$t("main.chargingPlan.precondition.summaryAll"));
			} else if (this.precondition) {
				parts.push(
					this.$t("main.chargingPlan.precondition.summary", {
						precondition: this.fmtDurationLong(this.precondition),
					})
				);
			}
			return parts.join(", ");
		},
		optimizationDescription(): string {
			if (this.disabled) {
				return this.$t("main.chargingPlan.optimization.disabledDescription");
			}
			const variant = this.localContinuous
				? this.isCo2
					? "continuousCo2"
					: "continuousPrice"
				: this.cheapestKey;
			return this.$t(`main.chargingPlan.optimization.${variant}Description`);
		},
		preconditionOptions() {
			const options = [QUARTER_HOUR, HALF_HOUR, ONE_HOUR, TWO_HOURS, EVERYTHING];

			// support custom values (via API)
			if (this.localPrecondition && !options.includes(this.localPrecondition)) {
				options.push(this.localPrecondition);
			}

			return options.map((s) => ({
				value: s,
				name:
					s === EVERYTHING
						? this.$t("main.chargingPlan.precondition.optionAll")
						: this.fmtDurationLong(s),
			}));
		},
	},
	watch: {
		precondition: {
			handler(newValue: number) {
				// Only update if value actually changed from external source
				if (newValue !== this.localPrecondition) {
					this.localPrecondition = newValue;
				}
			},
			immediate: true,
		},
		continuous: {
			handler(newValue: boolean) {
				// Only update if value actually changed from external source
				if (newValue !== this.localContinuous) {
					this.localContinuous = newValue;
				}
			},
			immediate: true,
		},
	},
	methods: {
		formId(name: string) {
			return `chargingplan-${this.id}-${name}`;
		},
		updateStrategy(): void {
			const strategy: PlanStrategy = {
				continuous: this.localContinuous,
				precondition: this.localPrecondition,
			};
			this.$emit("update", strategy);
		},
	},
});
</script>

<style scoped>
.icon {
	transform: rotate(0deg);
	transition: transform var(--evcc-transition-medium) ease;
}
.iconUp {
	transform: rotate(-180deg);
}
</style>
