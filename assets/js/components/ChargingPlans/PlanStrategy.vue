<template>
	<div class="collapsible-wrapper" :class="{ open: show }">
		<div class="collapsible-content pb-3">
			<div v-if="disabled" class="row mb-4">
				<div class="small text-muted">
					<strong class="text-primary">{{ $t("general.note") }}</strong>
					{{ $t("main.chargingPlan.strategyDisabledDescription") }}
				</div>
			</div>
			<div v-else class="row">
				<div class="col-12 col-sm-6 col-lg-3 offset-lg-3 mb-3">
					<div class="row">
						<label :for="formId('continuous')" class="col-form-label col-5 col-sm-12">
							{{ $t("main.chargingPlan.optimization.label") }}
						</label>
						<div class="col-7 col-sm-12">
							<select
								:id="formId('continuous')"
								v-model="localContinuous"
								class="form-select"
								@change="updateStrategy"
							>
								<option :value="false">
									{{ $t("main.chargingPlan.optimization.cheapest") }}
								</option>
								<option :value="true">
									{{ $t("main.chargingPlan.optimization.continuous") }}
								</option>
							</select>
						</div>
					</div>
				</div>
				<div class="col-sm-6 col-lg-3 mb-3">
					<div class="row">
						<label :for="formId('precondition')" class="col-form-label col-5 col-sm-12">
							{{ $t("main.chargingPlan.precondition.label") }}
						</label>
						<div class="col-7 col-sm-12">
							<select
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
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import formatter from "@/mixins/formatter";
import type { PlanStrategy } from "./types";

export default defineComponent({
	name: "ChargingPlanStrategy",
	mixins: [formatter],
	props: {
		id: [String, Number],
		show: Boolean,
		precondition: { type: Number, default: 0 },
		continuous: { type: Boolean, default: false },
		disabled: Boolean,
	},
	emits: ["update"],
	data() {
		return {
			localPrecondition: this.precondition,
			localContinuous: this.continuous,
		};
	},
	computed: {
		preconditionOptions() {
			const HOUR = 60 * 60;
			const QUARTER_HOUR = 0.25 * HOUR;
			const HALF_HOUR = 0.5 * HOUR;
			const ONE_HOUR = 1 * HOUR;
			const TWO_HOURS = 2 * HOUR;
			const EVERYTHING = 7 * 24 * HOUR;

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
