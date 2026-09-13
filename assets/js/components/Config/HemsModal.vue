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
						@click.prevent="activeTab = 'dynamic'"
					>
						{{ $t("config.hems.dynamicLimits") }}
					</a>
				</li>
				<li class="nav-item">
					<a
						class="nav-link"
						:class="{ active: activeTab === 'static' }"
						href="#"
						@click.prevent="activeTab = 'static'"
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
			<div data-testid="grid-export-limit">
				<div class="form-check form-switch">
					<input
						id="hemsExportLimitEnabled"
						class="form-check-input"
						type="checkbox"
						role="switch"
						:checked="exportLimitEnabled"
						:disabled="exportLimitState === 'saving'"
						@change="toggleExportLimit"
					/>
					<label for="hemsExportLimitEnabled" class="form-check-label">
						{{ $t("config.hems.exportLimit") }}
					</label>
					<div class="ps-2">
						<p class="text-muted small mb-2">
							{{ $t("config.hems.exportLimitHint") }}
						</p>
						<div class="collapsible-wrapper" :class="{ open: exportLimitEnabled }">
							<div class="collapsible-content ring-space">
								<div class="d-flex align-items-center gap-2">
									<div class="input-group input-width">
										<input
											id="hemsExportLimit"
											v-model.number="exportLimit"
											type="number"
											min="0"
											:aria-label="$t('config.hems.exportLimit')"
											aria-describedby="hemsExportLimitUnit"
											class="form-control text-end"
											@input="onExportLimitInput"
											@blur="commitExportLimit"
											@keydown.enter.prevent="commitExportLimit"
										/>
										<span id="hemsExportLimitUnit" class="input-group-text">
											W
										</span>
									</div>
									<SavingIndicator :state="exportLimitState" />
								</div>
								<div class="form-text evcc-gray mb-2">
									{{ $t("config.hems.exportLimitDescription") }}
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
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
import SavingIndicator, { type SavingState } from "../Helper/SavingIndicator.vue";
import formatter from "../../mixins/formatter";

// selector value for the relay variant; both variants save as type custom
const RELAY_OPTION = "relay";

let exportLimitTimer: ReturnType<typeof setTimeout> | undefined;

const initialValues = {
	type: ConfigType.Template,
	icon: undefined,
	deviceProduct: undefined,
	yaml: undefined,
	template: null,
};

export default defineComponent({
	name: "HemsModal",
	components: { DeviceModalBase, DownloadButton, SavingIndicator },
	mixins: [formatter],
	props: {
		yamlSource: String as PropType<YamlSource>,
		id: Number as PropType<number | undefined>,
	},
	emits: ["changed", "close"],
	data() {
		return {
			initialValues,
			sessions: [] as Array<{ created: string }>,
			changing: false,
			exportLimit: null as number | null,
			exportLimitEnabled: false,
			exportLimitState: null as SavingState,
			activeTab: "dynamic" as "dynamic" | "static",
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
		exportLimitUnchanged(): boolean {
			return (this.exportLimit || 0) === this.serverExportLimit;
		},
	},
	beforeUnmount() {
		clearTimeout(exportLimitTimer);
	},
	methods: {
		onOpen() {
			this.exportLimit = this.serverExportLimit || null;
			this.exportLimitEnabled = this.serverExportLimit > 0;
			this.exportLimitState = null;
			this.activeTab = "dynamic";
			this.loadSessions();
		},
		onExportLimitInput() {
			clearTimeout(exportLimitTimer);
			exportLimitTimer = setTimeout(this.commitExportLimit, 2000);
		},
		commitExportLimit() {
			clearTimeout(exportLimitTimer);
			if (this.exportLimitState === "saving" || this.exportLimitUnchanged) return;
			this.postExportLimit(this.exportLimit || 0);
		},
		toggleExportLimit($event: Event) {
			const { checked } = $event.target as HTMLInputElement;
			this.exportLimitEnabled = checked;
			this.exportLimitState = null;
			if (!checked) {
				this.exportLimit = null;
				this.commitExportLimit();
			}
		},
		async postExportLimit(value: number) {
			this.exportLimitState = "saving";
			try {
				await api.post(`gridexportlimit/${value}`);
				this.exportLimitState = "saved";
			} catch (e) {
				this.exportLimitState = null;
				console.error(e);
			}
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
.input-width {
	max-width: 180px;
}
</style>
