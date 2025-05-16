<template>
	<div class="mb-5 mb-lg-4" data-testid="plan-entry">
		<h5
			v-if="multiplePlans"
			class="d-flex gap-3 align-items-baseline d-lg-none mb-4 fw-normal evcc-gray"
			data-testid="repeating-plan-title"
		>
			<span class="text-uppercase fs-6">
				{{ `${$t("main.chargingPlan.planNumber", { number: "#1" })}` }}
			</span>
		</h5>

		<div class="row d-none d-lg-flex mb-2">
			<div v-if="multiplePlans" class="plan-id d-flex"></div>
			<div class="col-3">
				<label :for="formId('day')">
					{{ $t("main.chargingPlan.day") }}
				</label>
			</div>
			<div class="col-2">
				<label :for="formId('time')">
					{{ $t("main.chargingPlan.time") }}
				</label>
			</div>
			<div :class="showPrecondition ? 'col-2' : 'col-3'">
				<label :for="formId('goal')">
					{{ $t("main.chargingPlan.goal") }}
				</label>
			</div>
			<div v-if="showPrecondition" class="col-1">
				<label :for="formId('precondition')">
					{{ $t("main.chargingPlan.preconditionShort") }}
				</label>
			</div>
			<div class="col-1">
				<label :for="formId('active')"> {{ $t("main.chargingPlan.active") }} </label>
			</div>
		</div>
		<div class="row">
			<div
				v-if="multiplePlans"
				class="plan-id d-none d-lg-flex align-items-center justify-content-start fs-6"
			>
				#1
			</div>
			<div class="col-5 d-lg-none col-form-label">
				<label :for="formId('day')">
					{{ $t("main.chargingPlan.day") }}
				</label>
			</div>
			<div class="col-7 col-lg-3 mb-2 mb-lg-0">
				<select
					:id="formId('day')"
					v-model="selectedDay"
					class="form-select me-2"
					data-testid="static-plan-day"
					@change="preview()"
				>
					<option v-for="opt in dayOptions()" :key="opt.value" :value="opt.value">
						{{ opt.name }}
					</option>
				</select>
			</div>
			<div class="col-5 d-lg-none col-form-label">
				<label :for="formId('day')">
					{{ $t("main.chargingPlan.time") }}
				</label>
			</div>
			<div class="col-7 col-lg-2 mb-2 mb-lg-0">
				<input
					:id="formId('time')"
					v-model="selectedTime"
					type="time"
					class="form-control mx-0"
					:step="60 * 5"
					data-testid="static-plan-time"
					required
					@change="preview()"
				/>
			</div>
			<div class="col-5 d-lg-none col-form-label">
				<label :for="formId('goal')">
					{{ $t("main.chargingPlan.goal") }}
				</label>
			</div>
			<div :class="['col-7', showPrecondition ? 'col-lg-2' : 'col-lg-3', 'mb-2', 'mb-lg-0']">
				<select
					v-if="socBasedPlanning"
					:id="formId('goal')"
					v-model="selectedSoc"
					class="form-select mx-0"
					data-testid="static-plan-soc"
					@change="preview()"
				>
					<option v-for="opt in socOptions" :key="opt.value" :value="opt.value">
						{{ opt.name }}
					</option>
				</select>
				<select
					v-else
					:id="formId('goal')"
					v-model="selectedEnergy"
					class="form-select mx-0"
					data-testid="static-plan-energy"
					@change="preview()"
				>
					<option v-for="opt in energyOptions" :key="opt.energy" :value="opt.energy">
						{{ opt.text }}
					</option>
				</select>
			</div>
			<div v-if="showPrecondition" class="col-5 d-lg-none col-form-label">
				<label :for="formId('precondition')">
					{{ $t("main.chargingPlan.preconditionLong") }}
				</label>
			</div>
			<div
				v-if="showPrecondition"
				class="col-7 col-lg-1 mb-2 mb-lg-0 d-flex align-items-center"
			>
				<PreconditionSelect
					:id="formId('precondition')"
					v-model="selectedPrecondition"
					testid="static-plan-precondition"
				/>
			</div>
			<div class="col-5 d-lg-none col-form-label">
				<label :for="formId('active')">
					{{ $t("main.chargingPlan.active") }}
				</label>
			</div>
			<div class="col-3 col-lg-1 d-flex align-items-center">
				<div class="form-check form-switch my-1">
					<input
						:id="formId('active')"
						class="form-check-input"
						type="checkbox"
						role="switch"
						data-testid="static-plan-active"
						:checked="!isNew"
						:disabled="timeInThePast"
						tabindex="0"
						@change="toggle"
					/>
				</div>
			</div>
			<div
				class="col-4 col-lg-2 d-flex align-items-center justify-content-end justify-content-lg-start"
			>
				<button
					v-if="dataChanged && !isNew"
					type="button"
					class="btn btn-sm btn-outline-primary border-0 text-decoration-underline"
					data-testid="static-plan-apply"
					:disabled="timeInThePast"
					tabindex="0"
					@click="update"
				>
					{{ $t("main.chargingPlan.update") }}
				</button>
			</div>
		</div>
		<!-- Large screen precondition description -->
		<div :class="multiplePlans ? 'plan-id-insert' : ''">
			<PreconditionSelect
				:id="formId('precondition')"
				v-model="selectedPrecondition"
				testid="static-plan-precondition"
				description-lg-only
			/>
		</div>
		<p class="mb-0" data-testid="plan-entry-warnings">
			<span v-if="timeInThePast" class="d-block text-danger my-2">
				{{ $t("main.targetCharge.targetIsInThePast") }}
			</span>
		</p>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/checkmark";
import { distanceUnit } from "@/units";

import formatter from "@/mixins/formatter";
import { energyOptions } from "@/utils/energyOptions";
import { defineComponent } from "vue";
import PreconditionSelect from "./PreconditionSelect.vue";

const LAST_TARGET_TIME_KEY = "last_target_time";
const LAST_SOC_GOAL_KEY = "last_soc_goal";
const LAST_ENERGY_GOAL_KEY = "last_energy_goal";
const DEFAULT_TARGET_TIME = "7:00";

export default defineComponent({
	name: "ChargingPlanStaticSettings",
	components: { PreconditionSelect },
	mixins: [formatter],
	props: {
		id: [String, Number],
		soc: Number,
		energy: Number,
		time: Date,
		rangePerSoc: Number,
		socPerKwh: Number,
		capacity: Number,
		socBasedPlanning: Boolean,
		multiplePlans: Boolean,
		precondition: Number,
		showPrecondition: Boolean,
	},
	emits: ["static-plan-updated", "static-plan-removed", "plan-preview"],
	data() {
		return {
			selectedDay: null as string | null,
			selectedTime: null as string | null,
			selectedSoc: this.soc,
			selectedEnergy: this.energy,
			active: false,
			selectedPrecondition: this.precondition,
		};
	},
	computed: {
		selectedDate() {
			return new Date(`${this.selectedDay}T${this.selectedTime || "00:00"}`);
		},
		socOptions() {
			// a list of entries from 5 to 100 with a step of 5
			return Array.from(Array(20).keys())
				.map((i) => 5 + i * 5)
				.map(this.socOption);
		},
		energyOptions() {
			const options = energyOptions(
				0,
				this.capacity || 100,
				this.socPerKwh,
				this.fmtWh,
				this.fmtPercentage,
				"-"
			);
			// remove the first entry (0)
			return options.slice(1);
		},
		originalData() {
			if (this.isNew) {
				return {};
			}
			const t = this.time || new Date();
			return {
				soc: this.soc,
				energy: this.energy,
				day: this.fmtDayString(t),
				time: this.fmtTimeString(t),
				precondition: this.precondition,
			};
		},
		dataChanged() {
			const dateChanged =
				this.originalData.day != this.selectedDay ||
				this.originalData.time != this.selectedTime;
			const goalChanged = this.socBasedPlanning
				? this.originalData.soc != this.selectedSoc
				: this.originalData.energy != this.selectedEnergy;
			const preconditionChanged = this.originalData.precondition != this.selectedPrecondition;
			return dateChanged || goalChanged || preconditionChanged;
		},
		isNew() {
			return !this.time && (!this.soc || !this.energy);
		},
		timeInThePast() {
			const now = new Date();
			return now >= this.selectedDate;
		},
	},
	watch: {
		time() {
			this.initInputFields();
		},
		soc() {
			if (this.soc) {
				this.selectedSoc = this.soc;
			}
		},
		energy() {
			if (this.energy) {
				this.selectedEnergy = this.energy;
			}
		},
		isNew(value) {
			this.active = !value;
		},
		precondition(value) {
			this.selectedPrecondition = value;
		},
		selectedPrecondition() {
			this.preview();
		},
	},
	mounted() {
		this.initInputFields();
		this.preview();
	},
	methods: {
		formId(name: string) {
			return `chargingplan-${this.id}-${name}`;
		},
		socOption(value: number) {
			const name = this.fmtSocOption(value, this.rangePerSoc, distanceUnit());
			return { value, name };
		},
		initInputFields() {
			if (!this.selectedSoc) {
				this.selectedSoc = window.localStorage[LAST_SOC_GOAL_KEY] || 100;
			}
			if (!this.selectedEnergy) {
				this.selectedEnergy =
					window.localStorage[LAST_ENERGY_GOAL_KEY] || this.capacity || 10;
			}

			let t = this.time;
			if (!t) {
				// no time but existing selection, keep it
				if (this.selectedDay && this.selectedTime) {
					return;
				}
				t = this.defaultTime();
			}
			const date = new Date(t);
			this.selectedDay = this.fmtDayString(date);
			this.selectedTime = this.fmtTimeString(date);
		},
		dayOptions() {
			const options = [];
			const date = new Date();
			const labels = [
				this.$t("main.targetCharge.today"),
				this.$t("main.targetCharge.tomorrow"),
			];
			for (let i = 0; i < 7; i++) {
				const dayNumber = date.toLocaleDateString(this.$i18n?.locale, {
					day: "numeric",
					month: "short",
				});
				const dayName =
					labels[i] || date.toLocaleDateString(this.$i18n?.locale, { weekday: "short" });
				options.push({
					value: this.fmtDayString(date),
					name: `${dayNumber} (${dayName})`,
				});
				date.setDate(date.getDate() + 1);
			}
			return options;
		},
		update() {
			try {
				const hours = this.selectedDate.getHours();
				const minutes = this.selectedDate.getMinutes();
				window.localStorage[LAST_TARGET_TIME_KEY] = `${hours}:${minutes}`;
				if (this.selectedSoc) {
					window.localStorage[LAST_SOC_GOAL_KEY] = this.selectedSoc;
				}
				if (this.selectedEnergy) {
					window.localStorage[LAST_ENERGY_GOAL_KEY] = this.selectedEnergy;
				}
			} catch (e) {
				console.warn(e);
			}
			this.$emit("static-plan-updated", {
				time: this.selectedDate,
				soc: this.selectedSoc,
				energy: this.selectedEnergy,
				precondition: this.selectedPrecondition,
			});
		},
		preview(force = false) {
			if (!this.isNew && !force) {
				return;
			}
			this.$emit("plan-preview", {
				time: this.selectedDate,
				soc: this.selectedSoc,
				energy: this.selectedEnergy,
				precondition: this.selectedPrecondition,
			});
		},
		toggle(e: Event) {
			const { checked } = e.target as HTMLInputElement;
			if (checked) {
				this.update();
			} else {
				this.$emit("static-plan-removed");
				this.preview(true);
			}
			this.active = checked;
		},
		defaultTime() {
			const [hours, minutes] = (
				window.localStorage[LAST_TARGET_TIME_KEY] || DEFAULT_TARGET_TIME
			).split(":");

			const target = new Date();
			target.setSeconds(0);
			target.setMinutes(minutes);
			target.setHours(hours);
			// today or tomorrow?
			const isInPast = target < new Date();
			if (isInPast) {
				target.setDate(target.getDate() + 1);
			}
			return target;
		},
	},
});
</script>
<style scoped>
.plan-id-insert {
	margin-left: 2.5rem;
}
.plan-id {
	width: 2.5rem;
	color: var(--evcc-gray);
}
</style>
