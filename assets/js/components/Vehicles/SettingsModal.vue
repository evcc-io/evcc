<template>
	<GenericModal
		id="vehicleSettingsModal"
		ref="modal"
		size="lg"
		:title="$t('main.vehicleSettings.title')"
		data-testid="vehicle-settings-modal"
		@open="modalVisible"
		@closed="modalInvisible"
	>
		<div v-if="isModalVisible">
			<p class="text-gray mt-0 mb-4">
				{{ $t("main.vehicleSettings.description") }}
				<a :href="docsUrl" target="_blank" rel="noopener">
					{{ $t("main.vehicleSettings.learnMore") }}
				</a>
			</p>
			<div
				v-for="(vehicle, index) in vehicles"
				:key="vehicle.name"
				role="group"
				:aria-label="vehicle.title"
				:class="{ 'border-top pt-4': index > 0 }"
				class="mb-4"
			>
				<div class="d-flex flex-wrap align-items-center column-gap-2 row-gap-1 mb-3">
					<VehicleIcon :name="vehicle.icon" class="flex-shrink-0" />
					<strong class="text-truncate">{{ vehicle.title }}</strong>
					<Badge v-if="connectedLoadpoint(vehicle)" class="flex-shrink-0">
						{{
							$t("main.vehicleSettings.connectedTo", [
								connectedLoadpoint(vehicle)?.title ||
									$t("main.loadpoint.fallbackName"),
							])
						}}
					</Badge>
					<Badge v-else variant="muted" class="flex-shrink-0">
						{{ $t("main.vehicleSettings.notConnected") }}
					</Badge>
				</div>
				<SettingsFormRow
					:id="fieldId(vehicle, 'mode')"
					:label="$t('main.vehicleSettings.mode')"
				>
					<select
						:id="fieldId(vehicle, 'mode')"
						class="form-select form-select-sm"
						:value="vehicle.mode ?? ''"
						@change="changeMode(vehicle, $event)"
					>
						<option v-for="opt in modeOptions" :key="opt.value" :value="opt.value">
							{{ opt.name }}
						</option>
					</select>
				</SettingsFormRow>
				<template v-if="socSupported(vehicle)">
					<SettingsFormRow
						:id="fieldId(vehicle, 'limitSoc')"
						:label="$t('main.vehicleSettings.limitSoc')"
					>
						<select
							:id="fieldId(vehicle, 'limitSoc')"
							class="form-select form-select-sm"
							:value="vehicle.limitSoc ?? 0"
							@change="changeLimitSoc(vehicle, $event)"
						>
							<option
								v-for="opt in socOptions(vehicle)"
								:key="opt.value"
								:value="opt.value"
							>
								{{ opt.name }}
							</option>
						</select>
					</SettingsFormRow>
					<SettingsFormRow
						:id="fieldId(vehicle, 'minSoc')"
						:label="$t('main.vehicleSettings.minSoc')"
						:description="$t('main.vehicleSettings.minSocDescription')"
					>
						<select
							:id="fieldId(vehicle, 'minSoc')"
							class="form-select form-select-sm"
							:value="vehicle.minSoc ?? 0"
							@change="changeMinSoc(vehicle, $event)"
						>
							<option
								v-for="opt in socOptions(vehicle)"
								:key="opt.value"
								:value="opt.value"
							>
								{{ opt.name }}
							</option>
						</select>
					</SettingsFormRow>
				</template>
			</div>
			<p class="mb-0 border-top pt-4">
				<i18n-t keypath="main.vehicleSettings.editHint" tag="span" scope="global">
					<router-link to="/config#vehicles" @click="closeModal">
						{{ $t("config.main.title") }}
					</router-link>
				</i18n-t>
			</p>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import Badge from "../Helper/Badge.vue";
import GenericModal from "../Helper/GenericModal.vue";
import SettingsFormRow from "../Helper/SettingsFormRow.vue";
import VehicleIcon from "../VehicleIcon";
import api from "@/api";
import formatter from "@/mixins/formatter";
import { docsPrefix } from "@/i18n";
import { distanceUnit } from "@/units";
import { vehicleHasSoc, vehicleNotReachable } from "@/uiLoadpoints";
import { CHARGE_MODE, type SelectOption, type UiLoadpoint, type Vehicle } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";

const { OFF, PV, MINPV, NOW } = CHARGE_MODE;

export default defineComponent({
	name: "VehicleSettingsModal",
	components: { Badge, GenericModal, SettingsFormRow, VehicleIcon },
	mixins: [formatter],
	props: {
		vehicles: { type: Array as PropType<Vehicle[]>, default: () => [] },
		loadpoints: { type: Array as PropType<UiLoadpoint[]>, default: () => [] },
	},
	data() {
		return {
			isModalVisible: false,
		};
	},
	computed: {
		docsUrl(): string {
			return `${docsPrefix()}/docs/features/limits`;
		},
		modeOptions(): SelectOption<string>[] {
			return [
				{ value: "", name: this.$t("main.vehicleSettings.keepAsIs") },
				...[OFF, PV, MINPV, NOW].map((mode) => ({
					value: mode,
					name: this.$t(`main.mode.${mode}`),
				})),
			];
		},
	},
	methods: {
		modalVisible(): void {
			this.isModalVisible = true;
		},
		modalInvisible(): void {
			this.isModalVisible = false;
		},
		closeModal(): void {
			(this.$refs["modal"] as InstanceType<typeof GenericModal> | undefined)?.close();
		},
		fieldId(vehicle: Vehicle, field: string): string {
			return `vehicleSettings-${vehicle.name}-${field}`;
		},
		connectedLoadpoint(vehicle: Vehicle): UiLoadpoint | undefined {
			return this.loadpoints.find((lp) => lp.vehicleName === vehicle.name && lp.connected);
		},
		socSupported(vehicle: Vehicle): boolean {
			const loadpoint = this.connectedLoadpoint(vehicle);
			if (loadpoint) {
				return loadpoint.socBasedCharging || loadpoint.vehicleNotReachable;
			}
			return vehicleHasSoc(vehicle) || vehicleNotReachable(vehicle);
		},
		socOptions(vehicle: Vehicle): SelectOption<number>[] {
			// 0 = none, then 5-100 in steps of 5
			const rangePerSoc = this.connectedLoadpoint(vehicle)?.rangePerSoc;
			return Array.from(Array(21).keys()).map((i) => {
				const soc = i * 5;
				return {
					value: soc,
					name:
						soc === 0
							? this.$t("general.none")
							: this.fmtSocOption(soc, rangePerSoc, distanceUnit()),
				};
			});
		},
		selectValue(event: Event): string {
			return (event.target as HTMLSelectElement).value;
		},
		changeMode(vehicle: Vehicle, event: Event): void {
			const mode = this.selectValue(event);
			if (mode === "") {
				api.delete(`vehicles/${vehicle.name}/mode`);
			} else {
				api.post(`vehicles/${vehicle.name}/mode/${mode}`);
			}
		},
		changeMinSoc(vehicle: Vehicle, event: Event): void {
			api.post(`vehicles/${vehicle.name}/minsoc/${this.selectValue(event)}`);
		},
		changeLimitSoc(vehicle: Vehicle, event: Event): void {
			api.post(`vehicles/${vehicle.name}/limitsoc/${this.selectValue(event)}`);
		},
	},
});
</script>
