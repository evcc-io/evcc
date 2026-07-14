<template>
	<GenericModal
		id="vehicleSettingsModal"
		ref="modal"
		size="lg"
		data-testid="vehicle-settings-modal"
		@open="modalVisible"
		@closed="modalInvisible"
	>
		<template #title>
			<i18n-t
				v-if="multipleVehicles"
				keypath="main.vehicleSettings.title"
				tag="span"
				scope="global"
			>
				<CustomSelect
					id="vehicleSettingsVehicle"
					inline
					:aria-label="$t('main.vehicle.changeVehicle')"
					:options="vehicleSelectOptions"
					:selected="vehicleName"
					@change="changeVehicle($event.target.value)"
				>
					<span class="vehicle-title">{{ vehicleTitle }}</span>
				</CustomSelect>
			</i18n-t>
			<span v-else>{{ $t("main.vehicleSettings.title", [vehicleTitle]) }}</span>
		</template>
		<div v-if="isModalVisible" class="container">
			<div class="mb-3 row">
				<label for="vehicleSettingsMode" class="col-sm-4 col-form-label pt-0 pt-sm-2">
					{{ $t("config.loadpoint.defaultModeLabel") }}
				</label>
				<div class="col-sm-8 col-lg-4 pe-0 d-flex align-items-center">
					<select
						id="vehicleSettingsMode"
						v-model="selectedMode"
						class="form-select form-select-sm"
						@change="changeMode"
					>
						<option v-for="opt in modeOptions" :key="opt.value" :value="opt.value">
							{{ opt.name }}
						</option>
					</select>
				</div>
				<div class="col-sm-8 offset-sm-4 mt-1">
					<small class="text-muted">{{ modeHelp }}</small>
				</div>
			</div>
			<template v-if="socSettingsVisible">
				<div class="mb-3 row">
					<label for="vehicleSettingsMinSoc" class="col-sm-4 col-form-label pt-0 pt-sm-2">
						{{ $t("main.loadpointSettings.minSoc.label") }}
					</label>
					<div class="col-sm-8 col-lg-4 pe-0 d-flex align-items-center">
						<select
							id="vehicleSettingsMinSoc"
							v-model.number="selectedMinSoc"
							class="form-select form-select-sm"
							@change="changeMinSoc"
						>
							<option
								v-for="soc in minSocOptions"
								:key="soc.value"
								:value="soc.value"
							>
								{{ soc.name }}
							</option>
						</select>
					</div>
					<div class="col-sm-8 offset-sm-4 mt-1">
						<small class="text-muted">
							{{
								$t("main.loadpointSettings.minSoc.description", [
									selectedMinSoc ? fmtPercentage(selectedMinSoc) : "x",
								])
							}}
						</small>
					</div>
				</div>
				<div class="mb-3 row">
					<label
						for="vehicleSettingsLimitSoc"
						class="col-sm-4 col-form-label pt-0 pt-sm-2"
					>
						{{ $t("main.loadpointSettings.limitSoc.label") }}
					</label>
					<div class="col-sm-8 col-lg-4 pe-0 d-flex align-items-center">
						<select
							id="vehicleSettingsLimitSoc"
							v-model.number="selectedLimitSoc"
							class="form-select form-select-sm"
							@change="changeLimitSoc"
						>
							<option
								v-for="soc in limitSocOptions"
								:key="soc.value"
								:value="soc.value"
							>
								{{ soc.name }}
							</option>
						</select>
					</div>
					<div class="col-sm-8 offset-sm-4 mt-1">
						<small class="text-muted">
							{{ $t("main.loadpointSettings.limitSoc.description") }}
						</small>
					</div>
				</div>
			</template>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import GenericModal from "../Helper/GenericModal.vue";
import CustomSelect from "../Helper/CustomSelect.vue";
import api from "@/api";
import formatter from "@/mixins/formatter";
import { distanceUnit } from "@/units";
import { vehicleHasSoc, vehicleNotReachable } from "@/uiLoadpoints";
import { CHARGE_MODE, type SelectOption, type UiLoadpoint, type Vehicle } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";

const { OFF, PV, MINPV, NOW } = CHARGE_MODE;

export default defineComponent({
	name: "VehicleSettingsModal",
	components: { GenericModal, CustomSelect },
	mixins: [formatter],
	props: {
		vehicles: { type: Array as PropType<Vehicle[]>, default: () => [] },
		loadpoints: { type: Array as PropType<UiLoadpoint[]>, default: () => [] },
	},
	data() {
		return {
			isModalVisible: false,
			vehicleName: "",
			selectedMinSoc: 0,
			selectedLimitSoc: 0,
			selectedMode: "",
		};
	},
	computed: {
		vehicle() {
			return this.vehicles.find((v) => v.name === this.vehicleName);
		},
		loadpoint() {
			return this.loadpoints.find((lp) => lp.vehicleName === this.vehicleName);
		},
		vehicleTitle(): string {
			return this.vehicle?.title || "";
		},
		multipleVehicles(): boolean {
			return this.vehicles.length > 1;
		},
		vehicleSelectOptions(): SelectOption<string>[] {
			return this.vehicles.map((v) => ({ value: v.name, name: v.title }));
		},
		vehicleMinSoc(): number {
			return this.vehicle?.minSoc ?? 0;
		},
		vehicleLimitSoc(): number {
			return this.vehicle?.limitSoc ?? 0;
		},
		vehicleMode(): string {
			return this.vehicle?.mode ?? "";
		},
		socSettingsVisible(): boolean {
			if (this.loadpoint) {
				return this.loadpoint.socBasedCharging || this.loadpoint.vehicleNotReachable;
			}
			return vehicleHasSoc(this.vehicle) || vehicleNotReachable(this.vehicle);
		},
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
		apiVehicle(): string {
			return `vehicles/${this.vehicleName}/`;
		},
	},
	watch: {
		vehicleMinSoc(value: number): void {
			this.selectedMinSoc = value;
		},
		vehicleLimitSoc(value: number): void {
			this.selectedLimitSoc = value;
		},
		vehicleMode(value: string): void {
			this.selectedMode = value;
		},
	},
	methods: {
		open(vehicleName: string) {
			this.vehicleName = vehicleName;
			const modalRef = this.$refs["modal"] as InstanceType<typeof GenericModal> | undefined;
			modalRef?.open();
		},
		changeVehicle(vehicleName: string) {
			this.vehicleName = vehicleName;
		},
		modalVisible(): void {
			this.isModalVisible = true;
		},
		modalInvisible(): void {
			this.isModalVisible = false;
		},
		socOption(soc: number): SelectOption<number> {
			return {
				value: soc,
				name:
					soc === 0
						? "---"
						: this.fmtSocOption(soc, this.loadpoint?.rangePerSoc, distanceUnit()),
			};
		},
		changeMinSoc(): void {
			api.post(`${this.apiVehicle}minsoc/${this.selectedMinSoc}`);
		},
		changeLimitSoc(): void {
			api.post(`${this.apiVehicle}limitsoc/${this.selectedLimitSoc}`);
		},
		changeMode(): void {
			if (this.selectedMode === "") {
				api.delete(`${this.apiVehicle}mode`);
			} else {
				api.post(`${this.apiVehicle}mode/${this.selectedMode}`);
			}
		},
	},
});
</script>
<style scoped>
.vehicle-title {
	text-decoration: underline;
	text-decoration-color: var(--evcc-gray);
}
</style>
