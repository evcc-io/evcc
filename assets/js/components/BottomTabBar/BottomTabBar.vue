<template>
	<nav class="bottom-tab-bar position-fixed start-0 end-0 bottom-0">
		<div class="container d-flex align-items-stretch">
			<router-link
				to="/"
				class="tab-item d-flex flex-column flex-md-row align-items-center justify-content-center gap-md-1 text-decoration-none position-relative"
				exact-active-class="active"
				data-testid="tab-charge"
			>
				<shopicon-regular-lightning class="tab-icon d-block"></shopicon-regular-lightning>
				<span
					class="tab-label fw-bold text-uppercase mt-1 mt-md-0 text-truncate text-center text-md-start"
					>{{ $t("tabBar.charge") }}</span
				>
			</router-link>

			<router-link
				to="/battery"
				class="tab-item d-flex flex-column flex-md-row align-items-center justify-content-center gap-md-1 text-decoration-none position-relative"
				active-class="active"
				data-testid="tab-battery"
			>
				<BatteryIcon
					class="tab-icon d-block"
					:soc="batterySoc || 0"
					:grid-charge="batteryGridChargeActive"
					:hold="batteryHold"
				/>
				<span
					class="tab-label fw-bold text-uppercase mt-1 mt-md-0 text-truncate text-center text-md-start"
					>{{ $t("tabBar.battery") }}</span
				>
			</router-link>

			<router-link
				to="/forecast"
				class="tab-item d-flex flex-column flex-md-row align-items-center justify-content-center gap-md-1 text-decoration-none position-relative"
				active-class="active"
				data-testid="tab-forecast"
			>
				<ForecastGraphIcon class="tab-icon d-block" />
				<span
					class="tab-label fw-bold text-uppercase mt-1 mt-md-0 text-truncate text-center text-md-start"
					>{{ $t("tabBar.forecast") }}</span
				>
			</router-link>

			<router-link
				to="/sessions"
				class="tab-item d-flex flex-column flex-md-row align-items-center justify-content-center gap-md-1 text-decoration-none position-relative"
				active-class="active"
				data-testid="tab-sessions"
			>
				<shopicon-regular-cablecharge
					class="tab-icon d-block"
				></shopicon-regular-cablecharge>
				<span
					class="tab-label fw-bold text-uppercase mt-1 mt-md-0 text-truncate text-center text-md-start"
					>{{ $t("tabBar.sessions") }}</span
				>
			</router-link>

			<div
				class="tab-item dropup d-flex flex-column flex-md-row align-items-center justify-content-center gap-md-1 position-relative"
				:class="{ active: moreActive }"
				data-testid="tab-more"
			>
				<button
					ref="moreButton"
					type="button"
					class="tab-more-button d-flex flex-column flex-md-row align-items-center justify-content-center gap-md-1 w-100 h-100 border-0 bg-transparent position-relative"
					data-bs-toggle="dropdown"
					data-bs-display="static"
					aria-expanded="false"
				>
					<span
						v-if="showRootBadge"
						class="tab-badge position-absolute rounded-circle"
						:class="badgeClass"
					></span>
					<MoreIcon class="tab-icon d-block" size="s" />
					<span
						class="tab-label fw-bold text-uppercase mt-1 mt-md-0 text-truncate text-center text-md-start"
						>{{ $t("tabBar.more") }}</span
					>
				</button>
				<ul class="dropdown-menu dropdown-menu-end">
					<li>
						<button type="button" class="dropdown-item" @click="openSettingsModal">
							{{ $t("settings.title") }}
						</button>
					</li>
					<li>
						<router-link class="dropdown-item" to="/config" active-class="active">
							<span
								v-if="showConfigBadge"
								class="d-inline-block p-1 rounded-circle"
								:class="badgeClass"
							></span>
							{{ $t("config.main.title") }}
						</router-link>
					</li>
					<li>
						<router-link class="dropdown-item" to="/log" active-class="active">
							{{ $t("log.title") }}
						</router-link>
					</li>
					<li v-if="optimizeAvailable">
						<router-link class="dropdown-menu" to="/optimize" active-class="active">
							Optimize
						</router-link>
					</li>
					<li><hr class="dropdown-divider" /></li>
					<template v-if="authorizationRequired">
						<li>
							<h6 class="dropdown-header">
								{{ $t("authProviders.authorizationRequired") }}
							</h6>
						</li>
						<li v-for="provider in providers" :key="provider.id">
							<button type="button" class="dropdown-item" @click="handleAuthRequired">
								<span
									class="d-inline-block p-1 rounded-circle border border-light bg-warning"
								></span>
								{{ provider.title }}
							</button>
						</li>
						<li><hr class="dropdown-divider" /></li>
					</template>
					<li>
						<button type="button" class="dropdown-item" @click="openHelpModal">
							{{ $t("header.needHelp") }}
						</button>
					</li>
					<li>
						<a class="dropdown-item d-flex" href="https://evcc.io/" target="_blank">
							<span>evcc.io</span>
							<shopicon-regular-newtab
								size="s"
								class="ms-2 external"
							></shopicon-regular-newtab>
						</a>
					</li>
					<li v-if="isApp">
						<button type="button" class="dropdown-item" @click="openNativeSettings">
							{{ $t("header.nativeSettings") }}
						</button>
					</li>
					<li v-if="showLogout">
						<button type="button" class="dropdown-item" @click="doLogout">
							{{ $t("header.logout") }}
						</button>
					</li>
				</ul>
			</div>
		</div>
	</nav>
</template>

<script lang="ts">
import Modal from "bootstrap/js/dist/modal";
import Dropdown from "bootstrap/js/dist/dropdown";
import "@h2d2/shopicons/es/regular/lightning";
import "@h2d2/shopicons/es/regular/cablecharge";
import ForecastGraphIcon from "../MaterialIcon/ForecastGraph.vue";
import MoreIcon from "../MaterialIcon/More.vue";
import "@h2d2/shopicons/es/regular/newtab";
import BatteryIcon from "../Energyflow/BatteryIcon.vue";
import { logout, isLoggedIn } from "../Auth/auth";
import { isApp, sendToApp } from "@/utils/native";
import { isUserConfigError } from "@/utils/fatal";
import { defineComponent, type PropType } from "vue";
import type { FatalError, Sponsor, EvOpt, AuthProviders, Battery } from "@/types/evcc";

export default defineComponent({
	name: "BottomTabBar",
	components: {
		BatteryIcon,
		ForecastGraphIcon,
		MoreIcon,
	},
	props: {
		battery: { type: Object as PropType<Battery> },
		batteryGridChargeActive: Boolean,
		batteryMode: String,
		forecast: Object,
		authProviders: { type: Object as PropType<AuthProviders>, default: () => ({}) },
		sponsor: { type: Object as PropType<Sponsor>, default: () => ({}) },
		fatal: { type: Array as PropType<FatalError[]>, default: () => [] },
		experimental: Boolean,
		evopt: { type: Object as PropType<EvOpt>, required: false },
	},
	data() {
		return {
			isApp: isApp(),
			dropdown: null as Dropdown | null,
		};
	},
	computed: {
		batterySoc() {
			return this.battery?.soc;
		},
		batteryHold() {
			return this.batteryMode === "hold";
		},
		batteryConfigured() {
			return this.battery && this.battery.devices.length > 0;
		},
		providers() {
			return Object.entries(this.authProviders)
				.filter(([, provider]) => !provider.authenticated)
				.map(([title, { authenticated, id }]) => ({
					title,
					authenticated,
					id,
				}));
		},
		authorizationRequired() {
			return this.providers.length > 0;
		},
		sponsorExpires(): boolean {
			return !!this.sponsor?.status?.expiresSoon;
		},
		showConfigBadge() {
			return this.sponsorExpires || isUserConfigError(this.fatal);
		},
		showRootBadge() {
			return this.authorizationRequired || this.showConfigBadge;
		},
		badgeClass() {
			if (this.fatal.length > 0) {
				return "bg-danger";
			}
			return "bg-warning";
		},
		forecastAvailable() {
			const { grid, solar, co2 } = this.forecast || {};
			return grid || solar || co2;
		},
		optimizeAvailable() {
			return !!this.evopt && this.experimental;
		},
		showLogout() {
			return isLoggedIn();
		},
		moreActive() {
			const mainTabs = ["/", "/battery", "/forecast", "/sessions"];
			return !mainTabs.includes(this.$route.path);
		},
	},
	mounted() {
		const el = this.$refs.moreButton as HTMLElement;
		if (el) {
			this.dropdown = new Dropdown(el);
		}
	},
	unmounted() {
		this.dropdown?.dispose();
	},
	methods: {
		handleAuthRequired() {
			this.$router.push({ path: "/config" });
		},
		openSettingsModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("globalSettingsModal") as HTMLElement
			);
			modal.show();
		},
		openHelpModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("helpModal") as HTMLElement
			);
			modal.show();
		},
		openNativeSettings() {
			sendToApp({ type: "settings" });
		},
		async doLogout() {
			await logout();
			this.$router.push({ path: "/" });
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

.tab-item {
	flex: 1 1 0;
	min-width: 0;
	padding: 6px 0;
	color: var(--evcc-gray);
	border-top: 2px solid transparent;
	touch-action: manipulation;
	-webkit-tap-highlight-color: transparent;
	transition:
		color var(--evcc-transition-very-fast),
		border-color var(--evcc-transition-very-fast);
}

.tab-item:hover {
	color: color-mix(in srgb, var(--evcc-gray) 70%, white);
}

.tab-item:active {
	color: color-mix(in srgb, var(--evcc-gray) 70%, black);
}

.tab-item.active {
	color: var(--bs-primary);
	border-top-color: var(--bs-primary);
}

.tab-icon {
	width: 24px;
	height: 24px;
}

.tab-label {
	font-size: 10px;
	line-height: 1.2;
}

.tab-more-button {
	padding: 6px 0;
	color: inherit;
	cursor: pointer;
	outline: none;
	-webkit-tap-highlight-color: transparent;
}

.tab-badge {
	top: 2px;
	right: calc(50% - 16px);
	width: 8px;
	height: 8px;
}

.dropdown-menu {
	/* above sticky, below modal */
	z-index: 1045 !important;
	border: 1px solid var(--evcc-gray-10);
	border-bottom: none;
	border-radius: 10px 10px 0 0;
	box-shadow: none;
	background: var(--evcc-box);
	bottom: calc(100% + 2px);
	right: 0;
}

.dropdown-item.active,
.dropdown-item.router-link-active {
	background-color: transparent;
	color: var(--bs-primary);
	border-left: 2px solid var(--bs-primary);
}

@media (--md-and-up) {
	.bottom-tab-bar {
		min-height: auto;
	}
	.tab-icon {
		width: 18px;
		height: 18px;
	}
	.tab-label {
		font-size: 11px;
	}
}

.external {
	width: 18px;
	height: 20px;
}
</style>
