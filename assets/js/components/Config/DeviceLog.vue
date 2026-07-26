<template>
	<div class="mt-4">
		<hr class="my-4" />
		<div class="d-flex justify-content-between align-items-baseline mb-2">
			<h6 class="mb-0">{{ $t("config.deviceLog.title") }}</h6>
			<router-link
				v-if="area"
				:to="{ path: '/log', query: { areas: area, level } }"
				class="btn btn-link btn-sm evcc-default-text p-0"
			>
				{{ $t("config.deviceLog.showAll") }}
			</router-link>
		</div>
		<div ref="scroller" class="logs overflow-auto p-2">
			<code
				v-if="lines.length"
				class="d-block evcc-default-text textarea--tiny"
				data-testid="device-log"
			>
				<div v-for="({ className, line }, i) in lines" :key="i" :class="className">
					{{ line }}
				</div>
			</code>
			<p v-else class="mb-0">{{ $t("config.deviceLog.empty") }}</p>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import api from "@/api";
import { formatLogEntry, type LogEntry } from "@/utils/log";
import type { DeviceType, Timeout } from "@/types/evcc";

const COUNT = 100;
const LEVEL = "trace";
const INTERVAL = 2000;

export default defineComponent({
	name: "DeviceLog",
	props: {
		id: { type: Number, required: true },
		deviceType: { type: String as PropType<DeviceType>, required: true },
	},
	data() {
		return {
			entries: [] as LogEntry[],
			timeout: null as Timeout,
		};
	},
	computed: {
		level(): string {
			return LEVEL;
		},
		lines() {
			return this.entries.map((entry) => ({
				className: `log-${entry.level || "none"}`,
				line: formatLogEntry(entry),
			}));
		},
		area(): string | undefined {
			return this.entries[this.entries.length - 1]?.area;
		},
	},
	mounted() {
		this.updateLog();
		this.timeout = setInterval(this.updateLog, INTERVAL);
	},
	unmounted() {
		if (this.timeout) {
			clearInterval(this.timeout);
			this.timeout = null;
		}
	},
	methods: {
		async updateLog() {
			try {
				const response = await api.get(
					`/config/devices/${this.deviceType}/${this.id}/log`,
					{ params: { level: LEVEL, count: COUNT } }
				);
				this.entries = response.data || [];
				this.$nextTick(this.scrollToBottom);
			} catch (e) {
				console.error(e);
			}
		},
		scrollToBottom() {
			const el = this.$refs["scroller"] as HTMLElement;
			if (el) {
				el.scrollTop = el.scrollHeight;
			}
		},
	},
});
</script>

<style scoped>
.logs {
	max-height: 12rem;
	border-radius: 0.5rem;
	background: var(--evcc-box);
}
</style>
