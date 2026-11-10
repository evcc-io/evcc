<template>
	<DeviceModalBase
		:id="id"
		ref="deviceModal"
		name="hems"
		device-type="hems"
		:modal-title="$t('config.hems.title')"
		:provide-template-options="provideTemplateOptions"
		:initial-values="initialValues"
		:is-yaml-input-type="isYamlInputType"
		:on-template-change="handleTemplateChange"
		:hide-template-fields="fromYaml"
		:show-main-content="!experimental || activeTab === 'dynamic'"
		:hide-delete="true"
		:hide-info="true"
		:keep-open-on-remove="true"
		@added="onAdded"
		@updated="onUpdated"
		@removed="onRemoved"
		@open="onOpen"
		@close="$emit('close')"
	>
		<template v-if="id !== undefined && !fromYaml" #template-action>
			<button
				type="button"
				class="btn btn-outline-secondary border-0"
				:aria-label="$t('config.general.change')"
				:title="$t('config.general.change')"
				:disabled="changing"
				@click="handleChange"
			>
				<shopicon-regular-edit size="s" class="flex-shrink-0"></shopicon-regular-edit>
			</button>
		</template>
		<template #pre-content>
			<p class="mt-0 mb-4">
				{{ $t("config.hems.description") }}
				<a :href="docsLink" target="_blank" rel="noopener">
					{{ $t("config.general.docsLink") }}
				</a>
			</p>
			<ul v-if="experimental" class="nav nav-tabs mb-4">
				<li class="nav-item">
					<a
						class="nav-link"
						:class="{ active: activeTab === 'dynamic' }"
						href="#"
						@click.prevent="selectTab('dynamic')"
					>
						{{ $t("config.hems.dynamicLimits") }}
					</a>
				</li>
				<li class="nav-item">
					<a
						class="nav-link"
						:class="{ active: activeTab === 'static' }"
						href="#"
						@click.prevent="selectTab('static')"
					>
						{{ $t("config.hems.staticLimits") }} 🧪
					</a>
				</li>
			</ul>
		</template>
		<template #description>
			<div v-if="configured" class="mb-4" data-testid="grid-sessions">
				<h6 class="mb-3">{{ $t("config.hems.recordedEvents") }}</h6>
				<div class="events-box rounded p-3">
					<p v-if="!sessionCount" class="mb-0 text-muted">
						{{ $t("config.hems.noEvents") }}
					</p>
					<template v-else>
						<p class="mb-3">
							{{ $t("config.hems.eventsRecorded", { count: sessionCount }) }}<br />
							<template v-if="lastEventTimeAgo">
								{{ $t("config.hems.lastEvent", { timeAgo: lastEventTimeAgo }) }}
							</template>
						</p>
						<DownloadButton :label="$t('general.download')" :href="downloadHref()" />
					</template>
				</div>
				<hr class="mt-4 mb-0" />
			</div>
			<p v-if="fromYaml" class="text-muted">
				{{ $t("config.general.fromYamlHint") }}
			</p>
		</template>
		<template v-if="experimental && activeTab === 'static'" #post-content>
			<StaticLimitSetting
				id="hemsExportLimit"
				data-testid="grid-export-limit"
				:label="$t('config.hems.exportLimit')"
				:hint="$t('config.hems.exportLimitHint')"
				:description="$t('config.hems.exportLimitDescription')"
				:value="serverExportLimit"
				endpoint="gridexportlimit"
			/>
			<StaticLimitSetting
				v-if="peakLimitAvailable"
				id="hemsPeakLimit"
				class="mt-4"
				data-testid="grid-peak-limit"
				:label="$t('config.hems.peakLimit')"
				:hint="$t('config.hems.peakLimitHint')"
				:description="$t('config.hems.peakLimitDescription')"
				:value="serverPeakLimit"
				endpoint="gridpeakbaseline"
			>
				<div v-if="serverPeakLimit > 0" class="my-4 d-flex flex-column gap-3">
					<div class="fw-bold" :class="peakStatusClass">
						{{ peakStatusTitle }}
					</div>
					<DayStrip :days="peakDays" />
					<i18n-t
						v-if="peakFirstExceeded"
						keypath="config.hems.peakLimitExceededDetail"
						tag="p"
						class="mb-0 small evcc-gray"
						scope="global"
					>
						<template #date>
							<router-link :to="energyDay(peakFirstExceeded)" class="evcc-gray">{{
								peakFirstExceededDate
							}}</router-link>
						</template>
						<template #power
							><strong>{{ peakPowerLabel }}</strong></template
						>
					</i18n-t>
					<p v-if="!hasCircuit" class="mb-0 small evcc-gray">
						{{ $t("config.hems.peakLimitCircuitHint") }}
						<a href="#" class="evcc-gray" @click.prevent="openCircuitsModal">
							{{ $t("config.hems.peakLimitCircuitHintLink") }}
						</a>
					</p>
				</div>
			</StaticLimitSetting>
			<p v-if="!country" class="small evcc-gray mt-4 mb-0">
				{{ $t("config.hems.countryHint") }}
				<a href="#" class="evcc-gray" @click.prevent="openSiteModal">
					{{ $t("config.hems.countryHintLink") }}
				</a>
			</p>
		</template>
	</DeviceModalBase>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import "@h2d2/shopicons/es/regular/edit";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import { ConfigType, type YamlSource } from "@/types/evcc";
import { type DeviceValues } from "./DeviceModal";
import { customTemplateOption, type TemplateGroup } from "./DeviceModal/TemplateSelector.vue";
import customHemsYaml from "./defaultYaml/customHems.yaml?raw";
import relayHemsYaml from "./defaultYaml/relayHems.yaml?raw";
import api from "../../api";
import store from "@/store";
import { docsPrefix } from "@/i18n";
import DownloadButton from "../Helper/DownloadButton.vue";
import StaticLimitSetting from "./StaticLimitSetting.vue";
import DayStrip, { type Day } from "../Helper/DayStrip.vue";
import formatter from "../../mixins/formatter";
import { getModal, openModal, replaceModal } from "@/configModal";

// selector value for the relay variant; both variants save as type custom
const RELAY_OPTION = "relay";

type Tab = "dynamic" | "static";

// countries billing the month's highest 15-min grid import
const PEAK_LIMIT_COUNTRIES = ["AT", "BE"];

const initialValues = {
	type: ConfigType.Template,
	icon: undefined,
	deviceProduct: undefined,
	yaml: undefined,
	template: null,
};

export default defineComponent({
	name: "HemsModal",
	components: { DeviceModalBase, DownloadButton, StaticLimitSetting, DayStrip },
	mixins: [formatter],
	props: {
		yamlSource: String as PropType<YamlSource>,
		hasCircuit: Boolean,
		id: Number as PropType<number | undefined>,
	},
	emits: ["changed", "close"],
	data() {
		return {
			initialValues,
			sessions: [] as Array<{ created: string }>,
			changing: false,
		};
	},
	computed: {
		fromYaml(): boolean {
			return this.yamlSource === "file";
		},
		configured(): boolean {
			return this.id !== undefined || this.fromYaml;
		},
		sessionCount(): number {
			return this.sessions.length;
		},
		lastEvent() {
			return this.sessions[0] ?? null;
		},
		lastEventTimeAgo(): string {
			const created = this.lastEvent?.created;
			if (!created) return "";
			const ms = new Date(created).getTime();
			if (!Number.isFinite(ms)) return "";
			return (this as any).fmtTimeAgo(ms - Date.now());
		},
		docsLink(): string {
			return `${docsPrefix()}/external-limit`;
		},
		experimental(): boolean {
			return store.state?.experimental === true;
		},
		serverExportLimit(): number {
			return store.state?.gridExportLimit || 0;
		},
		activeTab(): Tab {
			return getModal("hems")?.type === "static" ? "static" : "dynamic";
		},
		country(): string {
			return store.state?.country || "";
		},
		peakLimitAvailable(): boolean {
			// an active limit stays editable after a country change
			return PEAK_LIMIT_COUNTRIES.includes(this.country) || this.serverPeakLimit > 0;
		},
		serverPeakLimit(): number {
			return store.state?.gridPeakBaseline || 0;
		},
		peakStatusTitle(): string {
			return this.peakFirstExceeded
				? this.$t("config.hems.peakLimitExceeded")
				: this.$t("config.hems.peakLimitBelow");
		},
		peakStatusClass(): string {
			return this.peakFirstExceeded ? "text-warning" : "text-primary";
		},
		peakDayPowers(): (number | null)[] {
			return store.state?.gridPeakDays || [];
		},
		// day of month, 0 = not exceeded
		peakFirstExceeded(): number {
			return this.peakDayPowers.findIndex((power) => (power ?? 0) > this.serverPeakLimit) + 1;
		},
		peakDays(): Day[] {
			const now = new Date();
			const count = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
			return Array.from({ length: count }, (_, i): Day => {
				const day = i + 1;
				const date = this.monthDate(day);
				if (day > now.getDate()) return { date, state: "none" };
				const to = this.energyDay(day);
				const power = this.peakDayPowers[i];
				// no complete slot recorded, unknown rather than below the limit
				if (power == null) return { date, state: "none", to };
				const value = this.$t("config.hems.peakLimitDayMax", {
					power: this.fmtW(power, this.POWER_UNIT.KW),
				});
				const state = power > this.serverPeakLimit ? "warning" : "ok";
				return { date, state, value, to };
			});
		},
		peakFirstExceededDate(): string {
			const day = this.peakFirstExceeded;
			return day ? this.fmtDayMonthShort(this.monthDate(day)) : "";
		},
		peakPowerLabel(): string {
			return this.fmtW(store.state?.gridPeak || 0, this.POWER_UNIT.KW);
		},
	},
	methods: {
		onOpen() {
			this.loadSessions();
		},
		selectTab(tab: Tab) {
			replaceModal("hems", { type: tab === "static" ? tab : undefined });
		},
		openSiteModal() {
			openModal("site");
		},
		openCircuitsModal() {
			openModal("circuits");
		},
		monthDate(day: number): Date {
			const now = new Date();
			return new Date(now.getFullYear(), now.getMonth(), day);
		},
		energyDay(day: number) {
			const now = new Date();
			return {
				path: "/energy",
				query: { year: now.getFullYear(), month: now.getMonth() + 1, day },
				hash: "#energy-grid",
			};
		},
		downloadHref(): string {
			const params = new URLSearchParams({ lang: this.$i18n?.locale });
			return `./api/gridsessions?${params.toString()}`;
		},
		async loadSessions() {
			try {
				const response = await api.get("gridsessions", {
					validateStatus: (code: number) => [200, 404].includes(code),
				});
				this.sessions = response.data || [];
			} catch (e) {
				this.sessions = [];
				console.error(e);
			}
		},
		provideTemplateOptions(products: any[]): TemplateGroup[] {
			if (this.fromYaml) {
				return [];
			}
			return [
				{
					label: "generic",
					options: [
						customTemplateOption(this.$t("config.hems.type.custom")),
						customTemplateOption(this.$t("config.hems.type.relay"), RELAY_OPTION),
					],
				},
				{
					label: "integrations",
					options: products,
				},
			];
		},
		isYamlInputType(type: ConfigType): boolean {
			return type === ConfigType.Custom || (type as string) === RELAY_OPTION;
		},
		handleTemplateChange(value: string, values: DeviceValues) {
			if (this.isYamlInputType(value as ConfigType)) {
				values.type = ConfigType.Custom;
				values.yaml = value === RELAY_OPTION ? relayHemsYaml : customHemsYaml;
			}
		},
		onAdded(name: string) {
			this.$emit("changed", { action: "added", name });
		},
		onUpdated() {
			this.$emit("changed", { action: "updated" });
		},
		onRemoved() {
			this.$emit("changed", { action: "removed" });
		},
		async handleChange() {
			if (this.id === undefined) return;
			if (!window.confirm(this.$t("config.hems.changeConfirm"))) return;
			this.changing = true;
			try {
				await (this.$refs["deviceModal"] as any).remove();
			} finally {
				this.changing = false;
			}
		},
	},
});
</script>

<style scoped>
.events-box {
	background: var(--evcc-background);
}
</style>
