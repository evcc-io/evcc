<template>
	<Card :title="$t('batterySettings.usageTab')" :subtitle="chargeSubtitle">
		<div class="d-flex gap-3 mb-4" data-testid="battery-priority">
			<shopicon-regular-sun
				size="s"
				class="text-primary flex-shrink-0 mt-1"
			></shopicon-regular-sun>
			<div>
				<div class="fw-bold mb-1">{{ $t("battery.config.priorityTitle") }}</div>
				<i18n-t
					:keypath="
						selectedPrioritySoc > 0
							? 'battery.config.priority'
							: 'battery.config.priorityNone'
					"
					tag="p"
					class="mb-0"
					scope="global"
				>
					<template #soc>
						<InlineSocSelect
							id="batteryExpPriority"
							:options="priorityOptions"
							:selected="selectedPrioritySoc"
							:label="fmtSoc(selectedPrioritySoc)"
							nowrap
							@change="changePrioritySoc"
						/>
					</template>
				</i18n-t>
			</div>
		</div>

		<div class="d-flex gap-3 mb-4" data-testid="battery-buffer">
			<shopicon-regular-lightning
				size="s"
				class="text-primary flex-shrink-0 mt-1"
			></shopicon-regular-lightning>
			<div>
				<div class="fw-bold mb-1">{{ $t("battery.config.bufferTitle") }}</div>
				<i18n-t
					:keypath="
						selectedBufferSoc < 100
							? 'battery.config.buffer'
							: 'battery.config.bufferNone'
					"
					tag="p"
					class="mb-0"
					scope="global"
				>
					<template #soc>
						<InlineSocSelect
							id="batteryExpBuffer"
							:options="bufferOptions"
							:selected="selectedBufferSoc"
							:label="fmtSoc(selectedBufferSoc)"
							nowrap
							@change="changeBufferSoc"
						/>
					</template>
					<template #start>
						<InlineSocSelect
							id="batteryExpBufferStart"
							:options="bufferStartOptions"
							:selected="selectedBufferStartSoc"
							:label="selectedBufferStartName"
							@change="changeBufferStart"
						/>
					</template>
				</i18n-t>
			</div>
		</div>

		<template v-if="controllable">
			<hr class="my-3" />
			<div class="form-check form-switch">
				<input
					id="batteryExpDischarge"
					:checked="batteryDischargeControl"
					class="form-check-input"
					type="checkbox"
					role="switch"
					@change="changeDischargeControl"
				/>
				<label class="form-check-label" for="batteryExpDischarge">
					{{ $t("battery.config.discharge") }}
				</label>
			</div>

			<div class="border-top pt-3 mt-3">
				<label class="form-label d-block mb-2">
					{{ $t("batterySettings.optimizerSocGoal.title") }}
				</label>
				<PlansRepeatingSettings
					id="battery"
					:start-number="1"
					:plans="batteryOptimizerSocGoals || []"
					@updated="saveBatteryOptimizerSocGoals"
				/>
				<small class="d-block text-muted mt-2">
					{{ $t("batterySettings.optimizerSocGoal.hint") }}
				</small>
			</div>

			<div class="border-top pt-3 mt-3">
				<div class="form-check form-switch mb-3">
					<input
						id="batteryExpManualPAEnabled"
						:checked="optimizerManualPAEnabled"
						class="form-check-input"
						type="checkbox"
						role="switch"
						@change="changeOptimizerManualPAEnabled"
					/>
					<label class="form-check-label" for="batteryExpManualPAEnabled">
						{{ $t("batterySettings.optimizerPA.enable") }}
					</label>
				</div>
				<div class="row g-3 align-items-end">
					<div class="col-sm-6">
						<label class="form-label" for="batteryExpManualPA">
							{{ $t("batterySettings.optimizerPA.value") }}
						</label>
						<div class="input-group">
							<input
								id="batteryExpManualPA"
								v-model="selectedOptimizerManualPA"
								type="number"
								inputmode="decimal"
								step="0.001"
								class="form-control mx-0"
								:disabled="!optimizerManualPAEnabled"
								@change="changeOptimizerManualPA"
							/>
							<span class="input-group-text">
								{{ pricePerKWhUnit(currency) }}
							</span>
						</div>
					</div>
				</div>
				<small class="d-block text-muted mt-2">
					{{ $t("batterySettings.optimizerPA.hint") }}
				</small>
			</div>
		</template>
	</Card>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/lightning";
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import api from "@/api";
import { CURRENCY, type Battery } from "@/types/evcc";
import Card from "../Helper/Card.vue";
import InlineSocSelect from "./InlineSocSelect.vue";
import PlansRepeatingSettings from "../ChargingPlans/PlansRepeatingSettings.vue";
import type { RepeatingPlan } from "../ChargingPlans/types";

// Battery usage controls for the experimental page. The logic is intentionally duplicated
// from the classic BatteryUsageSettings.vue (slated for removal) so the two can diverge
// during the transition.
export default defineComponent({
	name: "BatteryConfigCard",
	components: { Card, InlineSocSelect, PlansRepeatingSettings },
	mixins: [formatter],
	props: {
		bufferSoc: { type: Number, default: 100 },
		prioritySoc: { type: Number, default: 0 },
		bufferStartSoc: { type: Number, default: 0 },
		batteryDischargeControl: Boolean,
		batteryOptimizerSocGoals: {
			type: Array as PropType<RepeatingPlan[]>,
			default: () => [],
		},
		optimizerManualPA: { type: [Number, null] as PropType<number | null>, default: null },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		battery: { type: Object as PropType<Battery> },
	},
	data() {
		return {
			selectedBufferSoc: 100,
			selectedPrioritySoc: 0,
			selectedBufferStartSoc: 0,
			selectedOptimizerManualPA: "",
			optimizerManualPAEnabled: false,
		};
	},
	computed: {
		chargeSubtitle(): string {
			return `${this.$t("battery.card.soc")} ${this.fmtSoc(this.batterySoc)}`;
		},
		batterySoc(): number {
			return this.battery?.soc ?? 0;
		},
		controllable(): boolean {
			return (this.battery?.devices ?? []).some(({ controllable }) => controllable);
		},
		priorityOptions() {
			const options = [];
			for (let i = 100; i >= 0; i -= 5) {
				const disabled =
					i > this.selectedBufferSoc &&
					!(this.selectedBufferSoc == this.selectedPrioritySoc);
				options.push({ value: i, name: this.fmtSoc(i), disabled });
			}
			return options;
		},
		bufferOptions() {
			const options = [];
			for (let i = 100; i >= 5; i -= 5) {
				options.push({
					value: i,
					name: this.fmtSoc(i),
					disabled: i < this.selectedPrioritySoc,
				});
			}
			return options;
		},
		bufferStartOptions() {
			const options = [];
			for (let i = 100; i >= this.selectedBufferSoc; i -= 5) {
				options.push({ value: i, name: this.getBufferStartName(i) });
			}
			options.push({ value: 0, name: this.getBufferStartName(0) });
			return options;
		},
		selectedBufferStartName(): string {
			return this.getBufferStartName(this.selectedBufferStartSoc);
		},
	},
	watch: {
		prioritySoc: {
			handler(soc) {
				this.selectedPrioritySoc = soc;
			},
			immediate: true,
		},
		bufferSoc: {
			handler(soc) {
				this.selectedBufferSoc = soc || 100;
			},
			immediate: true,
		},
		bufferStartSoc: {
			handler(soc) {
				this.selectedBufferStartSoc = soc;
			},
			immediate: true,
		},
		optimizerManualPA: {
			handler(value: number | null) {
				this.optimizerManualPAEnabled = value !== null && value !== undefined;
				if (value !== null && value !== undefined) {
					this.selectedOptimizerManualPA = String(
						value * this.pricePerKWhDisplayFactor(this.currency)
					);
				}
			},
			immediate: true,
		},
	},
	methods: {
		changePrioritySoc($event: Event) {
			const soc = parseInt(($event.target as HTMLInputElement).value, 10);
			if (soc > (this.bufferSoc || 100)) {
				this.saveBufferSoc(soc);
				if (soc > this.bufferStartSoc && this.bufferStartSoc > 0) {
					this.setBufferStartSoc(soc);
				}
			} else {
				this.savePrioritySoc(soc);
			}
		},
		changeBufferStart($event: Event) {
			this.setBufferStartSoc(parseInt(($event.target as HTMLInputElement).value, 10));
		},
		async changeBufferSoc($event: Event) {
			const soc = parseInt(($event.target as HTMLInputElement).value, 10);
			if (soc === 100) {
				await this.setBufferStartSoc(0);
			} else if (soc > this.selectedBufferStartSoc && this.selectedBufferStartSoc > 0) {
				await this.setBufferStartSoc(soc);
			}
			await this.saveBufferSoc(soc);
		},
		async setBufferStartSoc(soc: number) {
			this.selectedBufferStartSoc = soc;
			await this.saveBufferStartSoc(soc);
		},
		async savePrioritySoc(soc: number) {
			this.selectedPrioritySoc = soc;
			try {
				await api.post(`prioritysoc/${encodeURIComponent(soc)}`);
			} catch (err) {
				console.error(err);
			}
		},
		async saveBufferSoc(soc: number) {
			this.selectedBufferSoc = soc;
			try {
				await api.post(`buffersoc/${encodeURIComponent(soc)}`);
			} catch (err) {
				console.error(err);
			}
		},
		async saveBufferStartSoc(soc: number) {
			try {
				await api.post(`bufferstartsoc/${encodeURIComponent(soc)}`);
			} catch (err) {
				console.error(err);
			}
		},
		async changeDischargeControl(e: Event) {
			try {
				await api.post(
					`batterydischargecontrol/${(e.target as HTMLInputElement).checked ? "true" : "false"}`
				);
			} catch (err) {
				console.error(err);
			}
		},
		async saveBatteryOptimizerSocGoals(goals: RepeatingPlan[]) {
			try {
				await api.post("batteryoptimizersocgoal", goals);
			} catch (err) {
				console.error(err);
			}
		},
		async changeOptimizerManualPAEnabled(e: Event) {
			const enabled = (e.target as HTMLInputElement).checked;
			this.optimizerManualPAEnabled = enabled;
			try {
				if (!enabled) {
					await api.delete("optimizermanualpa");
					return;
				}
				await this.saveOptimizerManualPA();
			} catch (err) {
				console.error(err);
			}
		},
		async changeOptimizerManualPA() {
			try {
				await this.saveOptimizerManualPA();
			} catch (err) {
				console.error(err);
			}
		},
		async saveOptimizerManualPA() {
			if (!this.optimizerManualPAEnabled) {
				return;
			}
			const value = Number.parseFloat(this.selectedOptimizerManualPA);
			if (!Number.isFinite(value)) {
				return;
			}
			const baseValue = value / this.pricePerKWhDisplayFactor(this.currency);
			await api.post(`optimizermanualpa/${encodeURIComponent(baseValue)}`);
		},
		getBufferStartName(value: number) {
			const key = value === 0 ? "never" : value === 100 ? "full" : "above";
			return this.$t(`battery.config.bufferStart.${key}`, { soc: this.fmtSoc(value) });
		},
		fmtSoc(soc: number) {
			return this.fmtPercentage(soc);
		},
	},
});
</script>
