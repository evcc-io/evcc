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
		:hide-delete="true"
		:hide-info="true"
		:keep-open-on-remove="true"
		@added="onAdded"
		@updated="onUpdated"
		@removed="onRemoved"
		@open="loadSessions"
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
		<template #description>
			<p class="mt-0 mb-4">
				{{ $t("config.hems.description") }}
				<a :href="docsLink" target="_blank" rel="noopener">
					{{ $t("config.general.docsLink") }}
				</a>
			</p>
			<div
				v-if="sessionCount"
				class="alert alert-info my-4 d-flex justify-content-between align-items-start flex-wrap gap-2"
				role="alert"
				data-testid="grid-sessions"
			>
				<div class="d-flex flex-wrap">
					<span class="me-2">{{
						$t("config.hems.eventsRecorded", { count: sessionCount })
					}}</span>
					<span v-if="lastEventTimeAgo">{{
						$t("config.hems.lastEvent", { timeAgo: lastEventTimeAgo })
					}}</span>
				</div>
				<a
					:href="csvLink"
					download
					class="alert-link text-nowrap"
					@click="handleDownloadClick($event, csvLink)"
				>
					{{ $t("config.hems.downloadCsv") }}
				</a>
			</div>
			<p v-if="fromYaml" class="text-muted">
				{{ $t("config.general.fromYamlHint") }}
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
import api from "../../api";
import { docsPrefix } from "@/i18n";
import { handleDownloadClick } from "../../utils/native";
import formatter from "../../mixins/formatter";

const initialValues = {
	type: ConfigType.Template,
	icon: undefined,
	deviceProduct: undefined,
	yaml: undefined,
	template: null,
};

export default defineComponent({
	name: "HemsModal",
	components: { DeviceModalBase },
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
		};
	},
	computed: {
		fromYaml(): boolean {
			return this.yamlSource === "file";
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
		csvLink(): string {
			const params = new URLSearchParams({
				format: "csv",
				lang: this.$i18n?.locale,
			});
			return `./api/gridsessions?${params.toString()}`;
		},
		docsLink(): string {
			return `${docsPrefix()}/docs/features/external-control`;
		},
	},
	methods: {
		handleDownloadClick,
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
					options: [customTemplateOption(this.$t("config.hems.customOption"))],
				},
				{
					label: "integrations",
					options: products,
				},
			];
		},
		isYamlInputType(type: ConfigType): boolean {
			return type === ConfigType.Custom;
		},
		handleTemplateChange(e: Event, values: DeviceValues) {
			const value = (e.target as HTMLSelectElement).value as ConfigType;
			if (value === ConfigType.Custom) {
				values.type = ConfigType.Custom;
				values.yaml = customHemsYaml;
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
