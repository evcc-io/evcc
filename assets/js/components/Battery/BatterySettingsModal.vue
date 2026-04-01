<template>
	<GenericModal
		id="batterySettingsModal"
		:title="$t('batterySettings.modalTitle')"
		size="xl"
		data-testid="battery-settings-modal"
		@open="modalVisible"
		@closed="modalInvisible"
	>
		<ul v-if="gridChargePossible || batteryGridChargeLimit !== null" class="nav nav-tabs mb-4">
			<li class="nav-item">
				<a
					class="nav-link"
					:class="{ active: usageTabActive }"
					href="#"
					@click.prevent="showUsageTab"
				>
					{{ $t("batterySettings.usageTab") }}
				</a>
			</li>
			<li class="nav-item">
				<a
					class="nav-link"
					:class="{ active: gridTabActive }"
					href="#"
					@click.prevent="showGridTab"
				>
					{{ $t("batterySettings.gridChargeTab") }}
				</a>
			</li>
		</ul>

		<BatteryUsageSettings
			v-show="usageTabActive"
			:buffer-soc="bufferSoc"
			:priority-soc="prioritySoc"
			:buffer-start-soc="bufferStartSoc"
			:battery-discharge-control="batteryDischargeControl"
			:battery="battery"
		/>
		<SmartCostLimit
			v-if="isModalVisible"
			v-show="gridTabActive"
			:current-limit="batteryGridChargeLimit"
			:last-limit="lastSmartCostLimit"
			:smart-cost-type="smartCostType"
			:currency="currency"
			:tariff="gridChargeTariff"
			:possible="gridChargePossible"
		/>
	</GenericModal>
</template>

<script lang="ts">
import SmartCostLimit from "../Tariff/SmartCostLimit.vue";
import BatteryUsageSettings from "./BatteryUsageSettings.vue";
import GenericModal from "../Helper/GenericModal.vue";
import settings from "@/settings";
import { defineComponent, type PropType } from "vue";
import type { Battery, CURRENCY, Forecast } from "@/types/evcc";
import { SMART_COST_TYPE } from "@/types/evcc";

export default defineComponent({
	name: "BatterySettingsModal",
	components: { SmartCostLimit, BatteryUsageSettings, GenericModal },
	props: {
		bufferSoc: { type: Number, default: 100 },
		prioritySoc: { type: Number, default: 0 },
		bufferStartSoc: { type: Number, default: 0 },
		batteryDischargeControl: Boolean,
		battery: { type: Object as PropType<Battery> },
		batteryGridChargeLimit: { type: Number, default: null },
		smartCostAvailable: Boolean,
		smartCostType: String as PropType<SMART_COST_TYPE>,
		tariffGrid: Number,
		currency: String as PropType<CURRENCY>,
		forecast: Object as PropType<Forecast>,
	},
	data() {
		return {
			isModalVisible: false,
			gridTabActive: false,
		};
	},
	computed: {
		usageTabActive() {
			return !this.gridTabActive;
		},
		batteryDevices() {
			return this.battery?.devices ?? [];
		},
		controllable() {
			return this.batteryDevices.some(({ controllable }) => controllable);
		},
		gridChargePossible() {
			return this.controllable && this.isModalVisible && this.smartCostAvailable;
		},
		gridChargeTariff() {
			if (this.smartCostType === SMART_COST_TYPE.CO2) {
				return this.forecast?.co2;
			}
			return this.forecast?.grid;
		},
		lastSmartCostLimit() {
			return settings.lastBatterySmartCostLimit;
		},
	},
	watch: {
		batteryGridChargeLimit() {
			this.verifyTabs();
		},
		gridChargePossible() {
			this.verifyTabs();
		},
	},
	methods: {
		showGridTab() {
			this.gridTabActive = true;
		},
		showUsageTab() {
			this.gridTabActive = false;
		},
		verifyTabs() {
			if (
				this.gridTabActive &&
				!this.gridChargePossible &&
				this.batteryGridChargeLimit === null
			) {
				this.gridTabActive = false;
			}
		},
		modalVisible() {
			this.isModalVisible = true;
		},
		modalInvisible() {
			this.isModalVisible = false;
		},
	},
});
</script>
