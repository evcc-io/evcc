<template>
	<div v-if="plans.length > 0" class="mb-3">
		<!-- Amber Status Banner when Paused -->
		<div v-if="isPaused" class="d-flex justify-content-end mb-3">
			<div
				class="alert alert-warning d-flex align-items-center justify-content-between p-2 px-3 mb-0 rounded w-100"
				role="alert"
				data-testid="repeating-plan-paused-badge"
			>
				<div class="d-flex align-items-center gap-2 text-truncate">
					<span class="fs-6 flex-shrink-0" aria-hidden="true">⏸️</span>
					<span class="text-truncate fw-medium">{{ pausedUntilText }}</span>
				</div>
				<button
					type="button"
					class="btn btn-sm btn-outline-primary border-0 text-decoration-underline p-0 flex-shrink-0 ms-2 text-nowrap"
					data-testid="repeating-plan-resume"
					tabindex="0"
					:aria-label="$t('main.chargingPlan.resume')"
					@click="resume"
				>
					{{ $t("main.chargingPlan.resume") }}
				</button>
			</div>
		</div>

		<!-- Pause Button & Dropdown when not paused -->
		<div v-else class="d-flex justify-content-end mb-2">
			<div class="dropdown">
				<button
					id="pauseRepeatingPlansDropdown"
					ref="pauseDropdown"
					type="button"
					class="btn btn-sm btn-outline-secondary d-flex align-items-center gap-1"
					data-bs-toggle="dropdown"
					data-bs-auto-close="outside"
					aria-expanded="false"
					:aria-label="$t('main.chargingPlan.pause')"
					data-testid="repeating-plan-pause"
					tabindex="0"
				>
					<span aria-hidden="true">⏸</span>
					<span>{{ $t("main.chargingPlan.pause") }}</span>
				</button>
				<div
					class="dropdown-menu dropdown-menu-end p-2 shadow-sm pause-dropdown-menu"
					aria-labelledby="pauseRepeatingPlansDropdown"
				>
					<button
						type="button"
						class="dropdown-item py-2 rounded text-start"
						:class="{ active: lastPausePreset === 'tomorrow' }"
						data-testid="pause-preset-tomorrow"
						tabindex="0"
						@click="pausePreset('tomorrow')"
					>
						{{ $t("main.chargingPlan.presetTomorrow") }}
					</button>
					<button
						type="button"
						class="dropdown-item py-2 rounded text-start"
						:class="{ active: lastPausePreset === 'friday' }"
						data-testid="pause-preset-friday"
						tabindex="0"
						@click="pausePreset('friday')"
					>
						{{ $t("main.chargingPlan.presetFriday") }}
					</button>
					<button
						type="button"
						class="dropdown-item py-2 rounded text-start"
						:class="{ active: lastPausePreset === 'sunday' }"
						data-testid="pause-preset-sunday"
						tabindex="0"
						@click="pausePreset('sunday')"
					>
						{{ $t("main.chargingPlan.presetSunday") }}
					</button>
					<div class="dropdown-divider my-1"></div>
					<button
						type="button"
						class="dropdown-item py-2 rounded text-start"
						:class="{ active: lastPausePreset === '24h' }"
						data-testid="pause-preset-24h"
						tabindex="0"
						@click="pausePreset('24h')"
					>
						{{ $t("main.chargingPlan.preset24h") }}
					</button>
					<button
						type="button"
						class="dropdown-item py-2 rounded text-start"
						:class="{ active: lastPausePreset === '48h' }"
						data-testid="pause-preset-48h"
						tabindex="0"
						@click="pausePreset('48h')"
					>
						{{ $t("main.chargingPlan.preset48h") }}
					</button>
					<button
						type="button"
						class="dropdown-item py-2 rounded text-start"
						:class="{ active: lastPausePreset === '7d' }"
						data-testid="pause-preset-7d"
						tabindex="0"
						@click="pausePreset('7d')"
					>
						{{ $t("main.chargingPlan.preset7d") }}
					</button>
					<div class="dropdown-divider my-1"></div>
					<button
						type="button"
						class="dropdown-item py-2 rounded d-flex justify-content-between align-items-center text-start"
						data-testid="pause-preset-custom"
						tabindex="0"
						:aria-expanded="showCustomPicker"
						@click="toggleCustomPicker"
					>
						<span>{{ $t("main.chargingPlan.presetCustom") }}</span>
						<small class="text-muted ms-2" aria-hidden="true">{{
							showCustomPicker ? "▲" : "▼"
						}}</small>
					</button>
					<div v-if="showCustomPicker" class="p-2 pt-3 border-top mt-2">
						<div class="mb-2">
							<label
								:for="`${formIdPrefix}-custom-date`"
								class="form-label small text-muted mb-1"
								>{{ $t("main.chargingPlan.day") }}</label
							>
							<input
								:id="`${formIdPrefix}-custom-date`"
								v-model="customDate"
								type="date"
								class="form-control form-control-sm"
								:min="todayIso"
								data-testid="pause-custom-date"
								tabindex="0"
								:aria-label="$t('main.chargingPlan.day')"
								required
							/>
						</div>
						<div class="mb-2">
							<label
								:for="`${formIdPrefix}-custom-time`"
								class="form-label small text-muted mb-1"
								>{{ $t("main.chargingPlan.time") }}</label
							>
							<input
								:id="`${formIdPrefix}-custom-time`"
								v-model="customTime"
								type="time"
								class="form-control form-control-sm"
								data-testid="pause-custom-time"
								tabindex="0"
								:aria-label="$t('main.chargingPlan.time')"
								required
							/>
						</div>
						<button
							type="button"
							class="btn btn-sm btn-primary w-100 mt-2"
							:disabled="isCustomInvalid"
							data-testid="pause-custom-apply"
							tabindex="0"
							:aria-label="$t('main.chargingPlan.update')"
							@click="pauseCustom"
						>
							{{ $t("main.chargingPlan.update") }}
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>

	<div v-for="(plan, index) in plans" :key="index" data-testid="plan-entry">
		<div>
			<ChargingPlanRepeatingSettings
				:showHeader="index === 0"
				:number="index + 2"
				class="mb-4"
				:formIdPrefix="formIdPrefix"
				v-bind="plan"
				:rangePerSoc="rangePerSoc"
				@updated="updatePlan(index, $event)"
				@removed="removePlan(index)"
			/>
		</div>
	</div>
	<div class="d-flex align-items-center pb-4">
		<button
			type="button"
			class="d-flex btn btn-sm btn-outline-secondary btn-add-plan border-0 align-items-center gap-2 evcc-gray"
			data-testid="repeating-plan-add"
			tabindex="0"
			:aria-label="$t('main.chargingPlan.addRepeatingPlan')"
			@click="addPlan"
		>
			<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
			{{ $t("main.chargingPlan.addRepeatingPlan") }}
		</button>
	</div>
</template>

<script lang="ts">
import Dropdown from "bootstrap/js/dist/dropdown";
import PlanRepeatingSettings from "./PlanRepeatingSettings.vue";
import deepEqual from "@/utils/deepEqual";
import formatter from "@/mixins/formatter";
import api from "@/api";
import settings, { type PausePreset } from "@/settings";
import { defineComponent, type PropType } from "vue";
import type { RepeatingPlan, Vehicle } from "@/types/evcc";

const DEFAULT_WEEKDAYS = [1, 2, 3, 4, 5];
const DEFAULT_TARGET_TIME = "07:00";
const DEFAULT_TARGET_SOC = 80;

export default defineComponent({
	name: "ChargingPlansRepeatingSettings",
	components: {
		ChargingPlanRepeatingSettings: PlanRepeatingSettings,
	},
	mixins: [formatter],
	props: {
		id: [Number, String],
		rangePerSoc: Number,
		plans: { type: Array as PropType<RepeatingPlan[]>, default: () => [] },
		vehicle: Object as PropType<Vehicle>,
	},
	emits: ["updated"],
	data() {
		return {
			localPausedUntil: undefined as string | null | undefined,
			showCustomPicker: false,
			customDate: "",
			customTime: "10:00",
		};
	},
	computed: {
		formIdPrefix(): string {
			return `chargingplan-lp${this.id}`;
		},
		effectivePausedUntil(): string | null | undefined {
			if (this.localPausedUntil !== undefined) {
				return this.localPausedUntil;
			}
			return this.vehicle?.pausedUntil;
		},
		isPaused(): boolean {
			if (!this.effectivePausedUntil) {
				return false;
			}
			const date = new Date(this.effectivePausedUntil);
			return !isNaN(date.getTime()) && date.getTime() > Date.now();
		},
		pausedUntilText(): string {
			if (!this.effectivePausedUntil) {
				return "";
			}
			const date = new Date(this.effectivePausedUntil);
			if (isNaN(date.getTime())) {
				return "";
			}
			const now = new Date();
			const startOfToday = new Date(now).setHours(0, 0, 0, 0);
			const startOfTarget = new Date(date).setHours(0, 0, 0, 0);
			const daysDiff = Math.round((startOfTarget - startOfToday) / 86400000);

			let formatted: string;
			if (daysDiff > 7) {
				const isSameYear = date.getFullYear() === now.getFullYear();
				const datePart = isSameYear ? this.fmtDayMonth(date) : this.fmtDayMonthYear(date);
				const timePart = this.fmtHourMinute(date);
				formatted = `${datePart} ${timePart}`;
			} else {
				formatted = this.fmtDayTime(date);
			}
			return this.$t("main.chargingPlan.pausedUntil", { time: formatted });
		},
		todayIso(): string {
			return this.fmtDayString(new Date());
		},
		isCustomInvalid(): boolean {
			const target = this.parseCustomDateTime();
			if (!target) return true;
			return isNaN(target.getTime()) || target.getTime() <= Date.now();
		},
		lastPausePreset(): PausePreset | undefined {
			return settings.lastPausePreset;
		},
	},
	watch: {
		id() {
			this.localPausedUntil = undefined;
			this.showCustomPicker = false;
		},
		vehicle: {
			handler(newVehicle?: Vehicle, oldVehicle?: Vehicle) {
				if (newVehicle?.name !== oldVehicle?.name) {
					this.localPausedUntil = undefined;
					this.showCustomPicker = false;
				} else if (newVehicle?.pausedUntil !== oldVehicle?.pausedUntil) {
					this.localPausedUntil = newVehicle?.pausedUntil;
				}
			},
			deep: true,
		},
	},
	mounted() {
		this.initCustomPickerDefaults();
	},
	methods: {
		deepEqual,
		initCustomPickerDefaults(): void {
			const tomorrow = new Date();
			tomorrow.setDate(tomorrow.getDate() + 1);
			this.customDate = this.fmtDayString(tomorrow);
			this.customTime = "10:00";
		},
		toggleCustomPicker(): void {
			this.showCustomPicker = !this.showCustomPicker;
			if (this.showCustomPicker && !this.customDate) {
				this.initCustomPickerDefaults();
			}
		},
		parseCustomDateTime(): Date | null {
			if (!this.customDate || !this.customTime) return null;
			const dateParts = this.customDate.split("-").map(Number);
			const timeParts = this.customTime.split(":").map(Number);
			if (dateParts.length < 3 || timeParts.length < 2) return null;
			const [year, month, day] = dateParts;
			const [hours, minutes] = timeParts;
			if (
				year === undefined ||
				month === undefined ||
				day === undefined ||
				hours === undefined ||
				minutes === undefined ||
				isNaN(year) ||
				isNaN(month) ||
				isNaN(day) ||
				isNaN(hours) ||
				isNaN(minutes)
			) {
				return null;
			}
			return new Date(year, month - 1, day, hours, minutes, 0, 0);
		},
		closeDropdown(): void {
			const el = (this.$refs as Record<string, any>)["pauseDropdown"];
			if (el) {
				const dropdownInstance =
					Dropdown.getInstance(el) || Dropdown.getOrCreateInstance(el);
				dropdownInstance?.hide();
			}
		},
		getPresetDate(preset: PausePreset): Date {
			const now = new Date();
			switch (preset) {
				case "tomorrow": {
					const d = new Date(now);
					d.setDate(d.getDate() + 1);
					d.setHours(10, 0, 0, 0);
					return d;
				}
				case "friday": {
					const d = new Date(now);
					const currentDay = now.getDay(); // 0 = Sun, 5 = Fri
					const daysUntilFriday = (5 - currentDay + 7) % 7;
					d.setDate(now.getDate() + daysUntilFriday);
					d.setHours(18, 0, 0, 0);
					if (d.getTime() <= now.getTime()) {
						d.setDate(d.getDate() + 7);
					}
					return d;
				}
				case "sunday": {
					const d = new Date(now);
					const currentDay = now.getDay(); // 0 = Sun
					const daysUntilSunday = (0 - currentDay + 7) % 7;
					d.setDate(now.getDate() + daysUntilSunday);
					d.setHours(18, 0, 0, 0);
					if (d.getTime() <= now.getTime()) {
						d.setDate(d.getDate() + 7);
					}
					return d;
				}
				case "24h":
					return new Date(now.getTime() + 24 * 3600 * 1000);
				case "48h":
					return new Date(now.getTime() + 48 * 3600 * 1000);
				case "7d":
					return new Date(now.getTime() + 7 * 24 * 3600 * 1000);
			}
		},
		async pausePreset(preset: PausePreset): Promise<void> {
			settings.lastPausePreset = preset;
			const date = this.getPresetDate(preset);
			await this.pauseUntil(date);
		},
		async pauseCustom(): Promise<void> {
			if (this.isCustomInvalid) return;
			const target = this.parseCustomDateTime();
			if (target) {
				await this.pauseUntil(target);
			}
		},
		async pauseUntil(date: Date): Promise<void> {
			if (!this.vehicle?.name) {
				console.warn("Cannot pause repeating plans: vehicle name is not available.");
				return;
			}
			const iso = date.toISOString();
			this.localPausedUntil = iso;
			this.showCustomPicker = false;
			this.closeDropdown();
			try {
				await api.post(
					`vehicles/${this.vehicle.name}/plan/pause/${encodeURIComponent(iso)}`,
					null
				);
			} catch (e) {
				console.error("Failed to pause repeating plans:", e);
				this.localPausedUntil = this.vehicle?.pausedUntil ?? null;
			}
		},
		async resume(): Promise<void> {
			if (!this.vehicle?.name) {
				console.warn("Cannot resume repeating plans: vehicle name is not available.");
				return;
			}
			this.localPausedUntil = null;
			try {
				await api.delete(`vehicles/${this.vehicle.name}/plan/pause`);
			} catch (e) {
				console.error("Failed to resume repeating plans:", e);
				this.localPausedUntil = this.vehicle?.pausedUntil ?? null;
			}
		},
		addPlan(): void {
			const newPlan = {
				weekdays: DEFAULT_WEEKDAYS,
				time: DEFAULT_TARGET_TIME,
				soc: DEFAULT_TARGET_SOC,
				active: false,
				tz: this.timezone(),
			};

			// update the plan without storing non-applied changes from other plans
			const plans = [...this.plans]; // clone array
			plans.push(newPlan);
			this.updatePlans(plans);
		},
		updatePlan(index: number, plan: RepeatingPlan): void {
			const plans = [...this.plans]; // clone array
			plans.splice(index, 1, plan);
			this.updatePlans(plans);
		},
		updatePlans(plans: RepeatingPlan[]): void {
			this.$emit("updated", plans);
		},
		removePlan(index: number): void {
			const plans = [...this.plans]; // clone array
			plans.splice(index, 1);
			this.updatePlans(plans);
		},
	},
});
</script>

<style scoped>
.btn-add-plan {
	margin-left: -0.5rem;
}
.pause-dropdown-menu {
	min-width: 240px;
	border: 1px solid rgba(255, 255, 255, 0.25);
	box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, 0.35);
}
</style>
