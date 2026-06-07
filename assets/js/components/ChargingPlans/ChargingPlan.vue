<template>
	<div class="text-center">
		<LabelAndValue
			class="root flex-grow-1"
			:label="$t('main.chargingPlan.title')"
			:class="disabled ? 'opacity-25' : 'opacity-100'"
			data-testid="charging-plan"
		>
			<div class="value m-0 d-block align-items-baseline justify-content-center">
				<button
					class="value-button p-0"
					:class="buttonColor"
					data-testid="charging-plan-button"
					@click="openModal"
				>
					<strong v-if="enabled">
						<span class="targetTimeLabel"> {{ targetTimeLabel }}</span>
						<div
							class="extraValue text-nowrap"
							:class="{ 'text-warning': planTimeUnreachable }"
						>
							{{ targetSocLabel }}
						</div>
					</strong>
					<span v-else class="text-decoration-underline">
						{{ $t("main.chargingPlan.none") }}
					</span>
				</button>
			</div>
		</LabelAndValue>
	</div>
</template>

<script lang="ts">
import LabelAndValue from "../Helper/LabelAndValue.vue";

import formatter from "@/mixins/formatter";
import minuteTicker from "@/mixins/minuteTicker";
import { optionStep, fmtEnergy } from "@/utils/energyOptions.ts";
import { defineComponent, type PropType } from "vue";
import type { CURRENCY, Vehicle } from "@/types/evcc";
import type { PlanStrategy } from "./types";
import type { Forecast } from "@/types/evcc.ts";

export default defineComponent({
	name: "ChargingPlan",
	components: {
		LabelAndValue,
	},
	mixins: [formatter, minuteTicker],
	props: {
		currency: String as PropType<CURRENCY>,
		disabled: Boolean,
		effectiveLimitSoc: Number,
		effectivePlanSoc: Number,
		effectivePlanTime: String,
		effectivePlanStrategy: Object as PropType<PlanStrategy>,
		id: [String, Number],
		limitEnergy: Number,
		mode: String,
		planActive: Boolean,
		planEnergy: Number,
		planTime: String,
		planTimeUnreachable: Boolean,
		planOverrun: Number,
		rangePerSoc: Number,
		smartCostType: String,
		socBasedPlanning: Boolean,
		socBasedCharging: Boolean,
		socPerKwh: Number,
		vehicle: Object as PropType<Vehicle>,
		capacity: Number,
		vehicleSoc: Number,
		vehicleLimitSoc: Number,
		vehicleNotReachable: Boolean,
		forecast: Object as PropType<Forecast>,
	},
	emits: ["open-modal"],
	data() {
		return {
			targetTimeLabel: "",
		};
	},
	computed: {
		buttonColor(): string {
			if (this.planTimeUnreachable) {
				return "text-warning";
			}
			if (!this.enabled) {
				return "text-gray";
			}
			return "evcc-default-text";
		},
		minSoc(): number | undefined {
			return this.vehicle?.minSoc;
		},
		limitSoc(): number | undefined {
			return this.vehicle?.limitSoc;
		},
		enabled(): boolean {
			return !!this.effectivePlanTime;
		},
		targetSocLabel(): string {
			if (this.socBasedPlanning && this.effectivePlanSoc) {
				return this.fmtPercentage(this.effectivePlanSoc);
			}
			return fmtEnergy(
				this.planEnergy,
				optionStep(this.capacity || 100),
				this.fmtWh,
				this.$t("main.targetEnergy.noLimit")
			);
		},
	},
	watch: {
		effectivePlanTime(): void {
			this.updateTargetTimeLabel();
		},
		everyMinute(): void {
			this.updateTargetTimeLabel();
		},
		"$i18n.locale": {
			handler(): void {
				this.updateTargetTimeLabel();
			},
		},
	},
	mounted(): void {
		this.updateTargetTimeLabel();
	},
	methods: {
		openModal(): void {
			this.$emit("open-modal");
		},
		updateTargetTimeLabel(): void {
			if (!this.effectivePlanTime) return;
			const targetDate = new Date(this.effectivePlanTime);
			this.targetTimeLabel = this.fmtAbsoluteDate(targetDate);
		},
	},
});
</script>

<style scoped>
.value {
	line-height: 1.2;
	border: none;
}
.value-button {
	font-size: 18px;
	border: none;
	background: none;
}
.root {
	transition: opacity var(--evcc-transition-medium) linear;
}
.value:hover {
	color: var(--bs-color-white);
}
.extraValue {
	color: var(--evcc-gray);
	font-size: 14px;
	text-decoration: none;
}
.targetTimeLabel {
	text-decoration: underline;
}
</style>
