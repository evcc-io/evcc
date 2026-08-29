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
				v-for="vehicle in vehicles"
				:key="vehicle.name"
				role="group"
				:aria-label="vehicle.title"
				class="d-flex gap-2 mb-3"
			>
				<VehicleIcon :name="vehicle.icon" class="flex-shrink-0" />
				<div class="flex-grow-1 overflow-hidden ring-space">
					<div class="d-flex flex-wrap align-items-center column-gap-2 row-gap-1 mb-3">
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
					<div class="row">
						<div class="col-6 col-lg-3 mb-3">
							<div class="small text-uppercase evcc-gray text-truncate">
								{{ $t("main.vehicleSettings.mode") }}
							</div>
							<CustomSelect
								:id="fieldId(vehicle, 'mode')"
								inline
								:options="modeOptions"
								:selected="vehicle.mode ?? ''"
								:ariaLabel="$t('main.vehicleSettings.mode')"
								@change="changeMode(vehicle, $event)"
							>
								<span class="text-decoration-underline text-primary">
									{{ optionName(modeOptions, vehicle.mode ?? "") }}
								</span>
							</CustomSelect>
						</div>
						<template v-if="socSupported(vehicle)">
							<div class="col-6 col-lg-3 mb-3">
								<div class="small text-uppercase evcc-gray text-truncate">
									{{ $t("main.vehicleSettings.minSoc") }}
								</div>
								<CustomSelect
									:id="fieldId(vehicle, 'minSoc')"
									inline
									:options="socOptions(vehicle, true)"
									:selected="vehicle.minSoc ?? 0"
									:ariaLabel="$t('main.vehicleSettings.minSoc')"
									@change="changeMinSoc(vehicle, $event)"
								>
									<span class="text-decoration-underline text-primary">
										{{
											optionName(
												socOptions(vehicle, true),
												vehicle.minSoc ?? 0
											)
										}}
									</span>
								</CustomSelect>
							</div>
							<div class="col-6 col-lg-3 mb-3">
								<div class="small text-uppercase evcc-gray text-truncate">
									{{ $t("main.vehicleSettings.limitSoc") }}
								</div>
								<CustomSelect
									:id="fieldId(vehicle, 'limitSoc')"
									inline
									:options="socOptions(vehicle)"
									:selected="vehicle.limitSoc || 100"
									:ariaLabel="$t('main.vehicleSettings.limitSoc')"
									@change="changeLimitSoc(vehicle, $event)"
								>
									<span class="text-decoration-underline text-primary">
										{{
											optionName(socOptions(vehicle), vehicle.limitSoc || 100)
										}}
									</span>
								</CustomSelect>
							</div>
						</template>
					</div>
				</div>
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
import CustomSelect from "../Helper/CustomSelect.vue";
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
	components: { Badge, CustomSelect, GenericModal, VehicleIcon },
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
			return `${docsPrefix()}/features/limits`;
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
		socOptions(vehicle: Vehicle, withNone = false): SelectOption<number>[] {
			// 5-100 in steps of 5, optionally preceded by 0 = none
			const rangePerSoc = this.connectedLoadpoint(vehicle)?.rangePerSoc;
			return Array.from(Array(21).keys())
				.filter((i) => withNone || i > 0)
				.map((i) => {
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
		optionName(options: SelectOption<string | number>[], value: string | number): string {
			return options.find((o) => o.value === value)?.name ?? String(value);
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
