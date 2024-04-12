<template>
	<div class="root safe-area-inset">
		<div class="container d-flex h-100 flex-column px-0 pb-4">
			<TopHeader showConfig :title="$t('log.title')" class="mx-4" />
			<div class="logs d-flex flex-column overflow-hidden flex-grow-1 px-4 mx-2 mx-sm-4">
				<div class="flex-grow-0 row py-4">
					<div class="col-6 col-lg-3 mb-4 mb-lg-0">
						<button
							type="button"
							class="btn text-nowrap d-flex w-100 w-lg-auto justify-content-between"
							:class="autoFollow ? 'btn-secondary' : 'btn-outline-secondary'"
							@click="toggleAutoFollow"
						>
							<span class="text-nowrap text-truncate">
								{{ $t("log.update") }}
							</span>
							<Record
								v-if="autoFollow"
								ref="spin"
								class="ms-1 spin flex-shrink-0"
								:style="{ animationDuration: `${updateInterval}ms` }"
							/>
							<Play v-else class="ms-1 flex-shrink-0" />
						</button>
					</div>
					<div class="col-6 offset-lg-1 col-lg-4 mb-4 mb-lg-0">
						<input
							type="search"
							class="form-control search"
							:placeholder="$t('log.search')"
							v-model="search"
						/>
					</div>
					<div class="filterLevel col-6 col-lg-2">
						<select
							class="form-select"
							v-model="level"
							:aria-label="$t('log.levelLabel')"
							@change="updateLogs()"
						>
							<option v-for="level in levels" :key="level" :value="level">
								{{ level }}
							</option>
						</select>
					</div>
					<div class="filterAreas col-6 col-lg-2">
						<select
							class="form-select"
							v-model="area"
							:aria-label="$t('log.areaLabel')"
							@focus="updateAreas()"
							@change="updateLogs()"
						>
							<option value="">{{ $t("log.areas") }}</option>
							<hr />
							<option v-for="area in areas" :key="area" :value="area">
								{{ area }}
							</option>
						</select>
					</div>
				</div>
				<hr class="my-0" />
				<div
					class="overflow-y-scroll pt-2 pb-4 flex-grow-1 d-flex flex-column"
					ref="log"
					@scroll="onScroll"
				>
					<code v-if="filteredLines.length" class="d-block evcc-default-text flex-grow-1">
						<div
							v-for="{ line, className, key } in lineEntries"
							:key="key"
							:class="className"
						>
							{{ line }}
						</div>
					</code>
					<p v-else class="my-4">{{ $t("log.noResults") }}</p>
					<div v-if="filteredLines.length" class="d-flex my-2 align-items-center">
						<div v-if="showMoreButton" class="d-flex align-items-center">
							<button
								class="btn btn-link btn-sm evcc-default-text px-0"
								type="button"
								@click="updateLogs(true)"
							>
								{{ $t("log.showAll") }}
							</button>
							<div class="m-2">|</div>
						</div>
						<a
							class="btn btn-link btn-sm evcc-default-text px-0"
							:href="downloadUrl"
							download
						>
							{{ $t("log.download") }}
						</a>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/arrowforward";
import TopHeader from "../components/TopHeader.vue";
import Play from "../components/MaterialIcon/Play.vue";
import Record from "../components/MaterialIcon/Record.vue";
import api from "../api";
import store from "../store";

const LEVELS = ["FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"];
const DEFAULT_LEVEL = "DEBUG";
const DEFAULT_COUNT = 1000;

const levelMatcher = new RegExp(`\\[.*?\\] (${LEVELS.join("|")})`);

export default {
	name: "Log",
	components: {
		TopHeader,
		Play,
		Record,
	},
	data() {
		return {
			lines: [],
			areas: [],
			search: "",
			level: DEFAULT_LEVEL,
			area: "",
			timeout: null,
			levels: LEVELS,
			busy: false,
		};
	},
	mounted() {
		this.startInterval();
		this.updateAreas();
	},
	unmounted() {
		this.stopInterval();
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

				const className = `log-${levelMatcher.exec(line)?.[1].toLowerCase() || "none"}`;

				return { key, className, line };
			});
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
			if (this.area) {
				params.append("area", this.area);
			}
			params.append("format", "txt");
			return `./api/log?${params.toString()}`;
		},
		autoFollow() {
			return this.timeout !== null;
		},
	},
	methods: {
		async updateLogs(showAll) {
			// prevent concurrent requests
			if (this.busy) return;

			try {
				this.busy = true;
				const response = await api.get("/log", {
					params: {
						level: this.level?.toLocaleLowerCase() || null,
						area: this.area || null,
						count: showAll ? null : DEFAULT_COUNT,
					},
				});
				this.lines = response.data?.result || [];
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
				const response = await api.get("/log/areas");
				this.areas = response.data?.result || [];
			} catch (e) {
				console.error(e);
			}
		},
		onScroll(e) {
			// disable follow when scrolling
			this.stopInterval();

			// start follow if scrolled to top and area is scrollable
			if (e.target.scrollTop === 0 && e.target.scrollHeight > e.target.clientHeight) {
				this.startInterval();
			}
		},
		toggleAutoFollow() {
			if (this.autoFollow) {
				this.stopInterval();
			} else {
				this.startInterval();
				this.$refs.log.scrollTo({ top: 0, behavior: "smooth" });
			}
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
.spin {
	animation: rotation 3s infinite ease-in-out;
}
@keyframes rotation {
	from {
		transform: rotate(0deg) scale(0.8, -0.8);
	}
	to {
		transform: rotate(1440deg) scale(0.8, -0.8);
	}
}
.log-warn {
	color: var(--bs-warning);
}
.log-error {
	color: var(--bs-danger);
}
</style>
