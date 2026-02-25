<template>
	<nav class="bottom-tab-bar position-fixed start-0 end-0 bottom-0">
		<div class="container d-flex align-items-stretch">
			<Item to="/" :label="$t('tabBar.charge')" data-testid="tab-charge" exact>
				<shopicon-regular-lightning class="tab-icon"></shopicon-regular-lightning>
			</Item>

			<Item to="/battery" :label="$t('tabBar.battery')" data-testid="tab-battery">
				<BatteryIcon
					class="tab-icon"
					:soc="batterySoc || 0"
					:grid-charge="batteryGridChargeActive"
					:hold="batteryHold"
				/>
			</Item>

			<Item to="/forecast" :label="$t('tabBar.forecast')" data-testid="tab-forecast">
				<ForecastGraphIcon class="tab-icon" />
			</Item>

			<Item to="/sessions" :label="$t('tabBar.sessions')" data-testid="tab-sessions">
				<shopicon-regular-cablecharge class="tab-icon"></shopicon-regular-cablecharge>
			</Item>

			<MoreMenu
				:auth-providers="authProviders"
				:sponsor="sponsor"
				:fatal="fatal"
				:experimental="experimental"
				:evopt="evopt"
			/>
		</div>
	</nav>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/lightning";
import "@h2d2/shopicons/es/regular/cablecharge";
import ForecastGraphIcon from "../MaterialIcon/ForecastGraph.vue";
import BatteryIcon from "../Energyflow/BatteryIcon.vue";
import Item from "./Item.vue";
import MoreMenu from "./MoreMenu.vue";
import { defineComponent, type PropType } from "vue";
import type { FatalError, Forecast, Sponsor, EvOpt, AuthProviders, Battery } from "@/types/evcc";

export default defineComponent({
	name: "BottomTabBar",
	components: {
		BatteryIcon,
		ForecastGraphIcon,
		Item,
		MoreMenu,
	},
	props: {
		battery: { type: Object as PropType<Battery> },
		batteryGridChargeActive: Boolean,
		batteryMode: { type: String as PropType<string> },
		forecast: { type: Object as PropType<Forecast> },
		authProviders: { type: Object as PropType<AuthProviders>, default: () => ({}) },
		sponsor: { type: Object as PropType<Sponsor>, default: () => ({}) },
		fatal: { type: Array as PropType<FatalError[]>, default: () => [] },
		experimental: Boolean,
		evopt: { type: Object as PropType<EvOpt>, required: false },
	},
	computed: {
		batterySoc() {
			return this.battery?.soc;
		},
		batteryHold() {
			return this.batteryMode === "hold";
		},
	},
});
</script>

<style scoped>
.bottom-tab-bar {
	z-index: 1030;
	min-height: 50px;
	padding-bottom: var(--safe-area-inset-bottom);
	background: color-mix(in srgb, var(--evcc-background) 80%, transparent);
	backdrop-filter: blur(20px);
	-webkit-backdrop-filter: blur(20px);
	border-top: 1px solid var(--evcc-gray-10);
}

@media (--md-and-up) {
	.bottom-tab-bar {
		min-height: auto;
	}
}
</style>
