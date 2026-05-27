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
				<div>
					<span>{{ $t("config.hems.eventsRecorded", { count: sessionCount }) }}</span>
					<span class="ms-2">{{
						$t("config.hems.lastEvent", { timeAgo: formatLastEvent(lastEvent.created) })
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

type HemsDevice = { id: number };

export default defineComponent({
	name: "HemsModal",
	components: { DeviceModalBase },
	mixins: [formatter],
	props: {
		yamlSource: String as PropType<YamlSource>,
	},
	emits: ["changed", "close"],
	data() {
		return {
			initialValues,
			id: undefined as number | undefined,
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
	created() {
		this.loadHemsId();
		this.loadSessions();
	},
	methods: {
		handleDownloadClick,
		async loadHemsId() {
			try {
				const response = await api.get("config/devices/hems");
				const devices = (response.data as HemsDevice[]) || [];
				this.id = devices[0]?.id;
			} catch (e) {
				console.error(e);
				this.id = undefined;
			}
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
		formatLastEvent(created: string): string {
			const diffMs = new Date().getTime() - new Date(created).getTime();
			return (this as any).fmtTimeAgo(-diffMs);
		},
		provideTemplateOptions(products: any[]): TemplateGroup[] {
			if (this.fromYaml) {
				return [];
			}
			return [
				{
					label: "generic",
					options: [customTemplateOption(this.$t("config.general.customOption"))],
				},
				{
					label: "providers",
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
		async onAdded(name: string) {
			await this.loadHemsId();
			this.$emit("changed", { action: "added", name });
		},
		onUpdated() {
			this.$emit("changed", { action: "updated" });
		},
		async onRemoved() {
			this.id = undefined;
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
