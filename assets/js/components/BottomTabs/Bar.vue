<template>
	<nav class="bottom-tab-bar d-flex position-fixed start-0 end-0 bottom-0">
		<div class="container d-flex align-items-stretch px-0">
			<Item to="/" :label="$t('tabBar.charge')" exact>
				<shopicon-regular-lightning class="tab-icon"></shopicon-regular-lightning>
			</Item>

			<Item to="/battery" :label="$t('tabBar.battery')">
				<BatteryIcon
					class="tab-icon"
					:soc="batterySoc || 0"
					:grid-charge="batteryGridChargeActive"
					:hold="batteryHold"
				/>
			</Item>

			<Item to="/forecast" :label="$t('tabBar.forecast')">
				<ForecastGraphIcon class="tab-icon" />
			</Item>

			<Item to="/sessions" :label="$t('tabBar.sessions')">
				<SessionsIcon class="tab-icon" />
			</Item>

			<MoreItem
				:active="moreActive"
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
import ForecastGraphIcon from "../MaterialIcon/ForecastGraph.vue";
import SessionsIcon from "../MaterialIcon/Sessions.vue";
import BatteryIcon from "../Energyflow/BatteryIcon.vue";
import Item from "./Item.vue";
import MoreItem from "./MoreItem.vue";
import { defineComponent, type PropType } from "vue";
import type { FatalError, Forecast, Sponsor, EvOpt, AuthProviders, Battery } from "@/types/evcc";

export default defineComponent({
	name: "BottomTabBar",
	components: {
		BatteryIcon,
		ForecastGraphIcon,
		SessionsIcon,
		Item,
		MoreItem,
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
		moreActive() {
			const mainTabs = ["/", "/battery", "/forecast", "/sessions"];
			return !mainTabs.includes(this.$route.path);
		},
	},
});
</script>

<style scoped>
.bottom-tab-bar {
	z-index: 1030;
	background: color-mix(in srgb, var(--tab-bar-background) 80%, transparent);
	backdrop-filter: blur(20px);
	-webkit-backdrop-filter: blur(20px);
	border-top: 1px solid var(--evcc-gray-10);
	box-shadow: 0 -1px 6px rgba(0, 0, 0, 0.05);
}

:root.dark .bottom-tab-bar {
	border-top-color: var(--bs-border-color);
}
</style>
