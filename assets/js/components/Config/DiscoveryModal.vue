<template>
	<GenericModal
		id="discoveryModal"
		title="Network discovery 🧪"
		config-modal-name="discovery"
		data-testid="discovery-modal"
		size="xl"
		@open="onOpen"
		@close="onClose"
	>
		<ul class="nav nav-tabs mb-4" role="tablist">
			<li v-for="tab in tabs" :key="tab.key" class="nav-item" role="presentation">
				<a
					:id="`discoveryTab-${tab.key}`"
					class="nav-link"
					:class="{ active: activeTab === tab.key }"
					href="#"
					role="tab"
					aria-controls="discoveryContent"
					:aria-selected="activeTab === tab.key"
					@click.prevent="activeTab = tab.key"
				>
					{{ tab.name }}
				</a>
			</li>
		</ul>
		<div id="discoveryContent" role="tabpanel" :aria-labelledby="`discoveryTab-${activeTab}`">
			<p>
				{{ description }}
				<a v-if="showShareLink" :href="shareUrl" target="_blank" rel="noopener">
					Share your result on GitHub.
				</a>
			</p>
			<textarea
				class="form-control font-monospace small"
				rows="20"
				wrap="off"
				readonly
				:aria-label="activeName"
				:value="content"
			></textarea>
			<div class="d-flex justify-content-between align-items-center gap-3 mt-1 small">
				<span v-if="loading" class="text-muted d-flex align-items-center gap-2">
					<span class="spinner-border spinner-border-sm" aria-hidden="true"></span>
					Scanning…
				</span>
				<a v-else href="#" @click.prevent="load(true)">Scan again</a>
				<CopyLink :text="markdown" class="mt-0" />
			</div>
			<p v-if="showWarning" class="mt-3 mb-0">
				<strong class="text-danger">Warning:</strong>
				Do not publish without checking the content.
			</p>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import CopyLink from "../Helper/CopyLink.vue";
import api from "@/api";

type Tab = "configured" | "all";

interface Report {
	system: string;
	devices: object[];
	hosts: object[];
}

// both scan passes with name lookup
const SCAN_DURATION = 10_000;

const SHARE_URL = "https://github.com/evcc-io/evcc/issues/34335";

const formatList = (items: object[], indent: string) =>
	items.length
		? `[\n${items.map((i) => `${indent}  ${JSON.stringify(i)}`).join(",\n")}\n${indent}]`
		: "[]";

const TABS: { key: Tab; name: string }[] = [
	{ key: "configured", name: "Configured devices" },
	{ key: "all", name: "All devices" },
];

export default defineComponent({
	name: "DiscoveryModal",
	components: { GenericModal, CopyLink },
	data() {
		return {
			isModalVisible: false,
			loading: false,
			activeTab: "configured" as Tab,
			report: null as Report | null,
			tabs: TABS,
			shareUrl: SHARE_URL,
		};
	},
	computed: {
		activeName(): string {
			return TABS.find((t) => t.key === this.activeTab)!.name;
		},
		count(): number {
			const list = this.activeTab === "all" ? this.report?.hosts : this.report?.devices;
			return list?.length ?? 0;
		},
		devices(): string {
			return `${this.count} ${this.count === 1 ? "device" : "devices"}`;
		},
		description(): string {
			return this.activeTab === "all"
				? `Complete result of the network scan: ${this.devices}.`
				: `Shows how the ${this.devices} configured by template appear in the local network. This data helps to improve the device suggestions. It contains no addresses or serial numbers.`;
		},
		showWarning(): boolean {
			return this.activeTab === "all";
		},
		showShareLink(): boolean {
			return this.activeTab === "configured";
		},
		// one object per line, key order as sent by the backend
		content(): string {
			if (!this.report) return "";
			const { system, devices, hosts } = this.report;
			if (this.activeTab === "all") return formatList(hosts, "");
			return `{\n  "system": ${JSON.stringify(system)},\n  "devices": ${formatList(devices, "  ")}\n}`;
		},
		// ready to paste into the GitHub issue
		markdown(): string {
			return "```json\n" + this.content + "\n```";
		},
	},
	methods: {
		onOpen() {
			this.isModalVisible = true;
			this.load();
		},
		onClose() {
			this.isModalVisible = false;
		},
		async load(refresh = false) {
			if (!refresh) {
				this.activeTab = "configured";
				this.report = null;
			}

			this.loading = true;
			await this.fetch(refresh);

			// pick up late answers of the second scan pass
			setTimeout(async () => {
				if (this.isModalVisible) await this.fetch();
				this.loading = false;
			}, SCAN_DURATION);
		},
		async fetch(refresh = false) {
			const res = await api.get("config/service/network/report", {
				params: refresh ? { refresh: 1 } : {},
			});
			this.report = res.data;
		},
	},
});
</script>
