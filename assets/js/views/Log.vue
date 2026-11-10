<template>
	<div class="root safe-area-inset">
		<div class="container d-flex h-100 flex-column px-0 pb-0 pb-md-3">
			<TopHeader :title="$t('log.title')" class="mx-4" />
			<div class="logs d-flex flex-column overflow-hidden flex-grow-1 px-4 mx-2 mx-sm-4">
				<div class="flex-grow-0 row py-4">
					<div class="col-6 col-lg-3 mb-4 mb-lg-0 d-flex gap-2">
						<div class="btn-group w-100 w-lg-auto d-flex">
							<button
								type="button"
								class="btn text-nowrap d-flex gap-2 flex-grow-1 text-nowrap text-truncate"
								:class="autoFollow ? 'btn-secondary' : 'btn-outline-secondary'"
								@click="toggleAutoFollow"
							>
								<span class="text-nowrap text-truncate">
									{{ $t("log.update") }}
								</span>
								<ProgressRing
									v-if="autoFollow"
									:key="tick"
									class="flex-shrink-0"
									:duration="updateInterval"
								/>
								<Play v-else class="flex-shrink-0 play" />
							</button>
							<a
								class="btn btn-outline-secondary flex-grow-0"
								:aria-label="$t('log.download')"
								:href="downloadUrl"
								download
								@click="handleDownloadClick($event, downloadUrl)"
							>
								<shopicon-regular-download
									size="s"
									class="icon d-flex"
								></shopicon-regular-download>
							</a>
						</div>
					</div>
					<div class="col-6 col-lg-3 mb-4 mb-lg-0">
						<input
							:value="q"
							@input="changeSearch"
							type="search"
							class="form-control search"
							:placeholder="$t('log.search')"
							data-testid="log-search"
						/>
					</div>
					<div class="filterLevel col-6 col-lg-3 d-flex joined">
						<select
							class="form-select"
							:aria-label="$t('log.levelLabel')"
							:value="level"
							@input="changeLevel"
						>
							<option
								v-for="l in levels"
								:key="l"
								:value="l"
								:class="{ stored: isStoredDefault(l) }"
							>
								{{ levelText(l) }}
							</option>
						</select>
						<button
							type="button"
							class="btn btn-outline-secondary flex-grow-0"
							:title="$t('log.saveDefault')"
							:aria-label="$t('log.saveDefault')"
							:disabled="!defaultChanged"
							@click="saveDefault"
						>
							<shopicon-regular-bookmark
								size="s"
								class="icon d-flex"
							></shopicon-regular-bookmark>
						</button>
					</div>
					<div class="filterAreas col-6 col-lg-3 d-flex joined">
						<MultiSelect
							id="logAreasSelect"
							class="flex-grow-1 areas-select"
							isTopLevel
							:modelValue="areas"
							:options="areaOptions"
							:selectAllLabel="$t('log.selectAll')"
							@update:model-value="changeAreas"
							@open="updateAreas()"
						>
							{{ areasLabel }}
						</MultiSelect>
						<button
							type="button"
							class="btn btn-outline-secondary flex-grow-0"
							:title="$t('log.saveAreas')"
							:aria-label="$t('log.saveAreas')"
							:disabled="!areasChanged"
							@click="saveAreas"
						>
							<shopicon-regular-bookmark
								size="s"
								class="icon d-flex"
							></shopicon-regular-bookmark>
						</button>
					</div>
				</div>
				<hr class="my-0" />
				<div
					ref="log"
					class="overflow-y-scroll pt-2 pb-4 flex-grow-1 d-flex flex-column"
					@scroll="onScroll"
				>
					<div v-if="showMoreButton" class="my-2">
						<button
							class="btn btn-link btn-sm evcc-default-text px-0"
							type="button"
							@click="updateLogs(true)"
						>
							{{ $t("log.showAll") }}
						</button>
					</div>
					<code
						v-if="filteredLines.length"
						class="d-block evcc-default-text flex-grow-1 textarea--tiny"
						data-testid="log-content"
						@copy="onCopy"
					>
						<div
							v-for="{ line, className, key } in lineEntries"
							:key="key"
							:class="className"
						>
							{{ line }}
						</div>
					</code>
					<p v-else class="my-4">{{ $t("log.noResults") }}</p>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/download";
import "@h2d2/shopicons/es/regular/bookmark";
import Header from "../components/Top/Header.vue";
import Play from "../components/MaterialIcon/Play.vue";
import ProgressRing from "../components/MaterialIcon/ProgressRing.vue";
import MultiSelect from "../components/Helper/MultiSelect.vue";
import api from "../api";
import store from "../store";
import { defineComponent, type PropType } from "vue";
import type { LogLevels, Timeout } from "@/types/evcc";
import { LOG_LEVELS, DEFAULT_LOG_LEVEL } from "@/utils/log";
import { handleDownloadClick } from "@/utils/native";
const DEFAULT_COUNT = 1000;

const sortedEntries = (levels: Record<string, string>) =>
	JSON.stringify(Object.entries(levels).sort(([a], [b]) => a.localeCompare(b)));

const levelMatcher = new RegExp(`\\[.*?\\] (${LOG_LEVELS.map((l) => l.toUpperCase()).join("|")})`);

export default defineComponent({
	name: "Log",
	components: {
		TopHeader: Header,
		Play,
		ProgressRing,
		MultiSelect,
	},
	props: {
		areas: { type: Array as PropType<string[]>, default: () => [] },
		level: { type: String, default: DEFAULT_LOG_LEVEL },
		q: { type: String, default: "" },
	},
	data() {
		return {
			lines: [] as string[],
			availableAreas: [] as string[],
			timeout: null as Timeout,
			levels: LOG_LEVELS,
			busy: false,
			tick: 0,
		};
	},
	head() {
		return { title: this.$t("log.title") };
	},
	computed: {
		filteredLines() {
			return this.lines.filter(
				(line) => !this.q || line.toLowerCase().includes(this.q.toLocaleLowerCase())
			);
		},
		lineEntries() {
			const occurrences = new Map();
			return this.filteredLines.map((line) => {
				// generate a unique key per line for performant dom updates
				let key = line.substring(0, 50);
				const count = occurrences.get(key) || 0;
				occurrences.set(key, count + 1);
				key = `${key}-${count + 1}`;

				const match = levelMatcher.exec(line)?.[1];
				const className = `log log-${match?.toLowerCase() || "none"}`;

				return { key, className, line };
			});
		},
		loadpointTitles(): Record<string, string> {
			return Object.fromEntries(
				(store.state?.loadpoints || []).map((lp, i) => [`lp-${i + 1}`, lp.title])
			);
		},
		areaOptions() {
			return this.availableAreas
				.slice()
				.sort((a, b) => a.localeCompare(b, undefined, { numeric: true }))
				.map((area) => ({
					name: this.areaName(area),
					value: area,
					badge: this.storedLevels[area]?.charAt(0),
					hint: this.storedLevels[area]
						? this.$t("log.storedLevel", { level: this.storedLevels[area] })
						: undefined,
				}));
		},
		storedLevels(): Record<string, string> {
			return store.state?.logLevels?.levels ?? {};
		},
		existingLevels(): Record<string, string> {
			// overrides of areas that no longer exist are dropped on the next write
			if (!this.availableAreas.length) return this.storedLevels;
			return Object.fromEntries(
				Object.entries(this.storedLevels).filter(([area]) =>
					this.availableAreas.includes(area)
				)
			);
		},
		storedDefault(): string {
			return store.state?.logLevels?.default ?? "INFO";
		},
		defaultChanged(): boolean {
			return this.level.toUpperCase() !== this.storedDefault;
		},
		nextLevels(): Record<string, string> {
			const level = this.level.toUpperCase();
			const levels = { ...this.existingLevels };
			this.areas.forEach((area) => {
				// an override equal to the default is redundant
				if (level === this.storedDefault) {
					delete levels[area];
				} else {
					levels[area] = level;
				}
			});
			return levels;
		},
		areasChanged(): boolean {
			return (
				this.areas.length > 0 &&
				sortedEntries(this.nextLevels) !== sortedEntries(this.existingLevels)
			);
		},
		areasLabel() {
			if (this.areas.length === 0) {
				return this.$t("log.areas");
			} else if (this.areas.length === 1) {
				return this.areaName(this.areas[0] as string);
			} else {
				return this.$t("log.nAreas", { count: this.areas.length });
			}
		},
		showMoreButton() {
			return this.lines.length === DEFAULT_COUNT;
		},
		updateInterval() {
			return (store.state?.interval || 10) * 1000;
		},
		downloadUrl() {
			const params = new URLSearchParams();
			if (this.level) {
				params.append("level", this.level);
			}
			this.areas.forEach((area) => {
				params.append("area", area);
			});
			params.append("format", "txt");
			return `./api/system/log?${params.toString()}`;
		},
		autoFollow() {
			return this.timeout !== null;
		},
	},
	watch: {
		areas() {
			this.updateLogs();
		},
		level() {
			this.updateLogs();
		},
	},
	mounted() {
		this.startInterval();
		this.updateAreas();
	},
	unmounted() {
		this.stopInterval();
	},
	methods: {
		handleDownloadClick,
		isStoredDefault(level: string) {
			return level.toUpperCase() === this.storedDefault;
		},
		levelText(level: string) {
			return this.isStoredDefault(level) ? `${level.toUpperCase()} ●` : level.toUpperCase();
		},
		areaName(area: string) {
			const title = this.loadpointTitles[area];
			return title ? `${area} (${title})` : area;
		},
		async updateLogs(showAll: boolean = false) {
			// prevent concurrent requests
			if (this.busy) return;

			this.tick++;
			try {
				this.busy = true;
				const response = await api.get("/system/log", {
					params: {
						level: this.level || null,
						area: this.areas.length ? this.areas : null,
						count: showAll ? null : DEFAULT_COUNT,
					},
				});
				this.lines = response.data || [];
				this.$nextTick(() => {
					if (showAll) {
						this.scrollToTop();
					} else {
						this.scrollToBottom();
					}
				});
			} catch (e) {
				console.error(e);
			}
			this.busy = false;
		},
		startInterval() {
			this.stopInterval();
			this.updateLogs();
			this.timeout = setInterval(() => {
				this.updateLogs();
			}, this.updateInterval);
		},
		stopInterval() {
			if (this.timeout) {
				clearTimeout(this.timeout);
				this.timeout = null;
			}
		},
		async saveLogLevels(logLevels: LogLevels) {
			try {
				await api.post("/config/logLevels", logLevels);
			} catch (e) {
				console.error(e);
			}
		},
		saveDefault() {
			return this.saveLogLevels({ default: this.level, levels: this.existingLevels });
		},
		saveAreas() {
			return this.saveLogLevels({ default: this.storedDefault, levels: this.nextLevels });
		},
		async updateAreas() {
			try {
				const response = await api.get("/system/log/areas");
				this.availableAreas = response.data || [];
			} catch (e) {
				console.error(e);
			}
		},
		onScroll(e: Event) {
			const t = e.target as HTMLElement;
			// disable follow when not at the bottom
			if (this.autoFollow && t && t.scrollTop + t.clientHeight < t.scrollHeight) {
				this.stopInterval();
			}
		},
		scrollToTop() {
			const log = this.$refs["log"] as HTMLElement;
			log.scrollTop = 0;
		},
		scrollToBottom() {
			const log = this.$refs["log"] as HTMLElement;
			log.scrollTop = log.scrollHeight;
		},
		toggleAutoFollow() {
			if (this.autoFollow) {
				this.stopInterval();
			} else {
				this.scrollToBottom();
				this.startInterval();
			}
		},
		updateQuery(
			{ level: l, areas: a, q: s }: { level?: string; areas?: string[]; q?: string },
			replace = false
		) {
			const newLevel = l || this.level;
			const newAreas = a || this.areas;
			const newSearch = s ?? this.q;

			// reset to default level
			const level = newLevel === DEFAULT_LOG_LEVEL ? undefined : newLevel;
			const areas = newAreas.length ? newAreas.join(",") : undefined;
			const q = newSearch || undefined;

			// typing shouldn't spam the history
			this.$router[replace ? "replace" : "push"]({ query: { level, areas, q } });
		},
		changeSearch(event: Event) {
			this.updateQuery({ q: (event.target as HTMLInputElement).value }, true);
		},
		changeLevel(event: Event) {
			this.updateQuery({ level: (event.target as HTMLSelectElement).value });
		},
		changeAreas(areas: string[]) {
			this.updateQuery({ areas });
		},
		onCopy(e: Event) {
			const selection = window.getSelection()?.toString();
			const event = e as ClipboardEvent;
			event.clipboardData?.setData("text/plain", "```\n" + selection + "\n```");
			event.preventDefault();
		},
	},
});
</script>
<style scoped>
.logs {
	border-radius: 2rem;
	background: var(--evcc-box);
}
.root {
	height: 100vh;
	height: 100dvh;
}
.btn {
	--bs-btn-border-width: 1px;
}
.stored {
	color: var(--evcc-darker-green);
}
.joined :deep(.form-select) {
	height: 100%;
	border-top-right-radius: 0;
	border-bottom-right-radius: 0;
}
.joined > .btn {
	border-top-left-radius: 0;
	border-bottom-left-radius: 0;
	margin-left: -1px;
}
.joined :deep(.form-select:focus),
.joined > .btn:hover {
	position: relative;
	z-index: 1;
}
.areas-select {
	min-width: 0;
}
.play {
	transform: scale(1.2);
}
@keyframes fadeIn {
	from {
		opacity: 0.3;
	}
	to {
		opacity: var(--opacity);
	}
}
.log {
	--opacity: 1;
	opacity: var(--opacity);
	animation-name: fadeIn;
	animation-duration: var(--transition-duration-fast);
	animation-fill-mode: forwards;
	animation-timing-function: ease-out;
	text-indent: 1rem hanging;
}

.log-warn {
	color: var(--bs-warning);
}
.log-error,
.log-fatal {
	color: var(--bs-danger);
}
.log-debug {
	--opacity: 0.7;
}
.log-trace {
	--opacity: 0.5;
}
</style>
