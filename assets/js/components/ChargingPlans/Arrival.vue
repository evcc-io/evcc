<template>
	<div class="mt-4 container">
		<div class="row mb-4">
			<label :for="formId('mode')" class="col-12 col-md-4 col-lg-3 col-form-label">
				{{ $t("config.loadpoint.defaultModeLabel") }}
			</label>
			<div class="col-12 col-md-8 col-lg-6">
				<select
					:id="formId('mode')"
					v-model="selectedMode"
					class="form-select"
					@change="changeMode"
				>
					<option v-for="opt in modeOptions" :key="opt.value" :value="opt.value">
						{{ opt.name }}
					</option>
				</select>
				<small class="form-text text-muted">{{ modeHelp }}</small>
			</div>
		</div>
		<template v-if="socSettingsVisible">
			<div class="row mb-4">
				<label :for="formId('minsoc')" class="col-12 col-md-4 col-lg-3 col-form-label">
					{{ $t("main.loadpointSettings.minSoc.label") }}
				</label>
				<div class="col-12 col-md-8 col-lg-6">
					<select
						:id="formId('minsoc')"
						v-model.number="selectedMinSoc"
						class="form-select"
						@change="changeMinSoc"
					>
						<option v-for="soc in minSocOptions" :key="soc.value" :value="soc.value">
							{{ soc.name }}
						</option>
					</select>
					<small class="form-text text-muted">
						{{
							$t("main.loadpointSettings.minSoc.description", [
								selectedMinSoc ? fmtPercentage(selectedMinSoc) : "x",
							])
						}}
					</small>
				</div>
			</div>
			<div class="row mb-4">
				<label :for="formId('limitsoc')" class="col-12 col-md-4 col-lg-3 col-form-label">
					{{ $t("main.loadpointSettings.limitSoc.label") }}
				</label>
				<div class="col-12 col-md-8 col-lg-6">
					<select
						:id="formId('limitsoc')"
						v-model.number="selectedLimitSoc"
						class="form-select"
						@change="changeLimitSoc"
					>
						<option v-for="soc in limitSocOptions" :key="soc.value" :value="soc.value">
							{{ soc.name }}
						</option>
					</select>
					<small class="form-text text-muted">
						{{ $t("main.loadpointSettings.limitSoc.description") }}
					</small>
				</div>
			</div>
		</template>
	</div>
</template>

<script lang="ts">
import { distanceUnit } from "@/units";
import formatter from "@/mixins/formatter";
import { CHARGE_MODE, type SelectOption } from "@/types/evcc";
import { defineComponent } from "vue";

const { OFF, PV, MINPV, NOW } = CHARGE_MODE;

export default defineComponent({
	name: "ChargingPlanArrival",
	mixins: [formatter],
	props: {
		id: [String, Number],
		minSoc: { type: Number, default: 0 },
		limitSoc: { type: Number, default: 0 },
		mode: { type: String, default: "" },
		vehicleName: String,
		vehicleNotReachable: Boolean,
		socBasedCharging: Boolean,
		rangePerSoc: Number,
	},
	emits: ["minsoc-updated", "limitsoc-updated", "mode-updated"],
	data() {
		return {
			selectedMinSoc: this.minSoc,
			selectedLimitSoc: this.limitSoc,
			selectedMode: this.mode,
		};
	},
	computed: {
		modeOptions(): SelectOption<string>[] {
			return [
				{ value: "", name: "---" },
				...[OFF, PV, MINPV, NOW].map((mode) => ({
					value: mode,
					name: this.$t(`main.mode.${mode}`),
				})),
			];
		},
		modeHelp(): string {
			return this.selectedMode === ""
				? this.$t("config.loadpoint.defaultModeHelpKeep")
				: this.$t("config.loadpoint.defaultModeHelp.charging");
		},
		minSocOptions(): SelectOption<number>[] {
			// a list of entries from 0 to 95 with a step of 5
			return Array.from(Array(20).keys())
				.map((i) => i * 5)
				.map(this.socOption);
		},
		limitSocOptions(): SelectOption<number>[] {
			// a list of entries from 0 to 100 with a step of 5
			return Array.from(Array(21).keys())
				.map((i) => i * 5)
				.map(this.socOption);
		},
		socSettingsVisible(): boolean {
			return this.socBasedCharging || this.vehicleNotReachable;
		},
	},
	watch: {
		minSoc(value: number): void {
			this.selectedMinSoc = value;
		},
		limitSoc(value: number): void {
			this.selectedLimitSoc = value;
		},
		mode(value: string): void {
			this.selectedMode = value;
		},
	},
	methods: {
		socOption(soc: number): SelectOption<number> {
			return {
				value: soc,
				name: soc === 0 ? "---" : this.fmtSocOption(soc, this.rangePerSoc, distanceUnit()),
			};
		},
		formId(name: string): string {
			return `chargingplan_${this.id}_${name}`;
		},
		changeMinSoc(): void {
			this.$emit("minsoc-updated", this.selectedMinSoc);
		},
		changeLimitSoc(): void {
			this.$emit("limitsoc-updated", this.selectedLimitSoc);
		},
		changeMode(): void {
			this.$emit("mode-updated", this.selectedMode);
		},
	},
});
</script>
