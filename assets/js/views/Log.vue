<template>
	<div class="root safe-area-inset">
		<div class="container d-flex h-100 flex-column px-0 pb-4">
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
								<Record
									v-if="autoFollow"
									ref="spin"
									class="spin flex-shrink-0"
									:style="{ animationDuration: `${updateInterval}ms` }"
								/>
								<Play v-else class="flex-shrink-0 play" />
							</button>
							<a
								class="btn btn-outline-secondary flex-grow-0"
								:aria-label="$t('log.download')"
								:href="downloadUrl"
								download
							>
								<shopicon-regular-download
									size="s"
									class="icon"
								></shopicon-regular-download>
							</a>
						</div>
					</div>
					<div class="col-6 offset-lg-1 col-lg-4 mb-4 mb-lg-0">
						<input
							v-model="search"
							type="search"
							class="form-control search"
							:placeholder="$t('log.search')"
							data-testid="log-search"
						/>
					</div>
					<div class="filterLevel col-6 col-lg-2">
						<select
							class="form-select"
							:aria-label="$t('log.levelLabel')"
							:value="level"
							@input="changeLevel"
						>
							<option v-for="l in levels" :key="l" :value="l">
								{{ l.toUpperCase() }}
							</option>
						</select>
					</div>
					<div class="filterAreas col-6 col-lg-2">
						<MultiSelect
							id="logAreasSelect"
							:modelValue="areas"
							:options="areaOptions"
							:selectAllLabel="$t('log.selectAll')"
							@update:model-value="changeAreas"
							@open="updateAreas()"
						>
							{{ areasLabel }}
						</MultiSelect>
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
						class="d-block evcc-default-text flex-grow-1"
						data-testid="log-content"
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

<script>
import "@h2d2/shopicons/es/regular/download";
import TopHeader from "../components/TopHeader.vue";
import Play from "../components/MaterialIcon/Play.vue";
import Record from "../components/MaterialIcon/Record.vue";
import MultiSelect from "../components/MultiSelect.vue";
import api from "../api";
import store from "../store";

const LEVELS = ["fatal", "error", "warn", "info", "debug", "trace"];
const DEFAULT_LEVEL = "debug";
const DEFAULT_COUNT = 1000;

const levelMatcher = new RegExp(`\\[.*?\\] (${LEVELS.map((l) => l.toUpperCase()).join("|")})`);

export default {
	name: "Log",
	components: {
		TopHeader,
		Play,
		Record,
		MultiSelect,
	},
	props: {
		areas: { type: Array, default: () => [] },
		level: { type: String, default: DEFAULT_LEVEL },
	},
	data() {
		return {
			lines: [],
			availableAreas: [],
			search: "",
			timeout: null,
			levels: LEVELS,
			busy: false,
		};
	},
	computed: {
		filteredLines() {
			return this.lines.filter(
				(line) =>
					!this.search || line.toLowerCase().includes(this.search.toLocaleLowerCase())
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

				const className = `log log-${levelMatcher.exec(line)?.[1].toLowerCase() || "none"}`;

				return { key, className, line };
			});
		},
		areaOptions() {
			return this.availableAreas.map((area) => ({ name: area, value: area }));
		},
		areasLabel() {
			if (this.areas.length === 0) {
				return this.$t("log.areas");
			} else if (this.areas.length === 1) {
				return this.areas[0];
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
		selectedAreas() {
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
		async updateLogs(showAll) {
			// prevent concurrent requests
			if (this.busy) return;

			try {
				this.busy = true;
				const response = await api.get("/system/log", {
					params: {
						level: this.level || null,
						area: this.areas.length ? this.areas : null,
						count: showAll ? null : DEFAULT_COUNT,
					},
				});
				this.lines = response.data?.result || [];
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
		async updateAreas() {
			try {
				const response = await api.get("/system/log/areas");
				this.availableAreas = response.data?.result || [];
			} catch (e) {
				console.error(e);
			}
		},
		onScroll(e) {
			// disable follow when not at the bottom
			if (
				this.autoFollow &&
				e.target.scrollTop + e.target.clientHeight < e.target.scrollHeight
			) {
				this.stopInterval();
			}
		},
		scrollToTop() {
			this.$refs.log.scrollTop = 0;
		},
		scrollToBottom() {
			this.$refs.log.scrollTop = this.$refs.log.scrollHeight;
		},
		toggleAutoFollow() {
			if (this.autoFollow) {
				this.stopInterval();
			} else {
				this.scrollToBottom();
				this.startInterval();
			}
		},
		updateQuery({ level, areas }) {
			let newLevel = level || this.level;
			let newAreas = areas || this.areas;
			// reset to default level
			if (newLevel === DEFAULT_LEVEL) newLevel = undefined;
			newAreas = newAreas.length ? newAreas.join(",") : undefined;
			this.$router.push({
				query: { level: newLevel, areas: newAreas },
			});
		},
		changeLevel(event) {
			this.updateQuery({ level: event.target.value });
		},
		changeAreas(areas) {
			this.updateQuery({ areas });
		},
	},
};
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
.play {
	transform: scale(1.2);
}
.spin {
	animation: rotation 3s infinite ease-in-out;
}
@keyframes rotation {
	from {
		transform: rotate(0deg) scale(1, -1);
	}
	to {
		transform: rotate(1440deg) scale(1, -1);
	}
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
	animation-duration: 1s;
	animation-fill-mode: forwards;
	animation-timing-function: ease-out;
	text-indent: 1rem hanging;
	/* smaller exception for mobile */
	font-size: 8px;
}
@media (min-width: 576px) {
	.log {
		/* default code size */
		font-size: 0.875em;
	}
}

.log-warn {
	color: var(--bs-warning);
}
.log-error,
.log-fatal {
	color: var(--bs-danger);
}
.log-debug {
	--opacity: 0.6;
}
.log-trace {
	--opacity: 0.4;
}
</style>
