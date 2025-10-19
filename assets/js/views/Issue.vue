<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader :title="$t('issue.title')" />
		<div class="row">
			<main class="col-12">
				<div class="mb-5">
					<p class="text-muted">{{ $t("issue.description") }}</p>
				</div>

				<!-- Help Type Selection -->
				<div class="mb-5">
					<h5 class="mb-3">{{ $t("issue.helpType.title") }}</h5>
					<div class="row g-3">
						<div class="col-12 col-md-6">
							<div class="form-check">
								<input
									id="helpTypeHelp"
									v-model="helpType"
									type="radio"
									value="discussion"
									class="form-check-input"
									name="helpType"
								/>
								<label for="helpTypeHelp" class="form-check-label">
									<strong>{{ $t("issue.helpType.discussion") }}</strong>
									<br />
									<small class="text-muted">
										{{ $t("issue.helpType.discussionDescription") }}
									</small>
								</label>
							</div>
						</div>
						<div class="col-12 col-md-6">
							<div class="form-check">
								<input
									id="helpTypeBug"
									v-model="helpType"
									type="radio"
									value="issue"
									class="form-check-input"
									name="helpType"
								/>
								<label for="helpTypeBug" class="form-check-label">
									<strong>{{ $t("issue.helpType.issue") }}</strong>
									<br />
									<small class="text-muted">
										{{ $t("issue.helpType.issueDescription") }}
									</small>
								</label>
							</div>
						</div>
					</div>
				</div>

				<form v-if="helpType" @submit.prevent="handleFormSubmit">
					<!-- Essential Form Section -->
					<div class="d-flex justify-content-between align-items-center mb-3">
						<h4>
							{{ $tt("issue.subTitle") }}
						</h4>
					</div>

					<p class="text-muted mb-4">
						🇬🇧 Please write your issue in English so everyone can participate.
					</p>

					<!-- Two Column Layout -->
					<div class="row mb-5 g-5">
						<!-- Left Column: Form Fields -->
						<div class="col-12 col-lg-6">
							<div class="mb-4">
								<label for="issueTitle" class="form-label">
									{{ $t("issue.issueTitle") }} *
								</label>
								<input
									id="issueTitle"
									v-model="issue.title"
									type="text"
									class="form-control"
									placeholder="Brief description of the problem"
									required
								/>
							</div>
							<div class="mb-4">
								<label for="issueDescription" class="form-label">
									{{ $t("issue.issueDescription") }} *
								</label>
								<textarea
									id="issueDescription"
									v-model="issue.description"
									class="form-control"
									rows="6"
									placeholder="Describe what you expected to happen and what actually happened..."
									required
								></textarea>
							</div>
							<div class="mb-4">
								<label for="stepsToReproduce" class="form-label">
									{{ $t("issue.stepsToReproduce") }} *
								</label>
								<textarea
									id="stepsToReproduce"
									v-model="issue.steps"
									class="form-control"
									rows="6"
									placeholder="1. Go to...&#10;2. Click on...&#10;3. See error..."
									required
								></textarea>
							</div>
							<div class="mb-4">
								<label for="version" class="form-label">
									{{ $t("issue.version") }}
								</label>
								<input
									id="version"
									v-model="versionString"
									type="text"
									class="form-control"
									required
									readonly
								/>
							</div>
							<div class="text-end">
								<small class="text-muted">* required</small>
							</div>
						</div>

						<!-- Right Column: Toggleable Sections -->
						<div class="col-12 col-lg-6">
							<div class="mb-4">
								<h5>{{ $t("issue.additional.title") }}</h5>
								<p class="text-muted small">
									{{ $t("issue.additional.description") }}
								</p>
							</div>

							<!-- Additional Items -->
							<IssueAdditionalItem
								id="issueYamlConfig"
								:included="sections.yamlConfig.included"
								:title="$t('issue.additional.yamlConfig')"
								:content="sections.yamlConfig.content"
								:helpType="helpType"
								@update:included="sections.yamlConfig.included = $event"
								@update:content="sections.yamlConfig.content = $event"
							>
								<template #description>
									<p class="text-muted small">
										{{ $t("issue.additional.yamlConfigDescription") }}<br />
										<span>
											{{ $t("issue.additional.source") }}:
											<code>{{ configPath || "---" }}</code>
										</span>
									</p>
								</template>
							</IssueAdditionalItem>

							<IssueAdditionalItem
								id="issueUiConfig"
								:included="sections.uiConfig.included"
								:title="$t('issue.additional.uiConfig')"
								:content="sections.uiConfig.content"
								:helpType="helpType"
								@update:included="sections.uiConfig.included = $event"
								@update:content="sections.uiConfig.content = $event"
							>
								<template #description>
									<p class="text-muted small">
										{{ $t("issue.additional.uiConfigDescription") }}<br />
										<span v-if="databasePath">
											{{ $t("issue.additional.source") }}:
											<code>{{ databasePath }}</code>
										</span>
									</p>
								</template>
							</IssueAdditionalItem>

							<!-- Logs Section with Special Controls -->
							<IssueAdditionalItem
								id="issueLogs"
								:included="sections.logs.included"
								:title="$t('issue.additional.logs')"
								:content="sections.logs.content"
								:helpType="helpType"
								@update:included="sections.logs.included = $event"
								@update:content="sections.logs.content = $event"
							>
								<template #description="{ openModal }">
									<p class="text-muted small">
										{{ $t("issue.additional.logsDescription") }}<br />
										Params:
										<button
											type="button"
											class="btn btn-link btn-sm p-0 text-decoration-underline small"
											@click="openModal"
										>
											{{ logLevel }}, {{ logCount }} lines,
											{{ logAreasLabel }}</button
										><br />
										{{ $t("issue.additional.source") }}:
										<router-link to="/log">{{ $t("log.title") }}</router-link>
									</p>
								</template>
								<template #controls>
									<div class="d-flex gap-3 mb-3">
										<div class="flex-shrink-0">
											<select
												id="logLevel"
												v-model="logLevel"
												class="form-select"
											>
												<option
													v-for="level in logLevels"
													:key="level"
													:value="level"
												>
													{{ level.toUpperCase() }}
												</option>
											</select>
										</div>
										<div class="flex-shrink-0">
											<div class="input-group log-lines-input">
												<input
													v-model.number="logCount"
													type="number"
													class="form-control text-end log-count-input"
													min="0"
													step="25"
												/>
												<span class="input-group-text">
													{{ $t("issue.additional.lines") }}
												</span>
											</div>
										</div>
										<div class="flex-grow-1">
											<MultiSelect
												v-model="logAreas"
												:options="logAreaOptions"
												:placeholder="$t('log.areas')"
												:selectAllLabel="$t('log.selectAll')"
											>
												{{ logAreasLabel }}
											</MultiSelect>
										</div>
									</div>
								</template>
							</IssueAdditionalItem>

							<IssueAdditionalItem
								id="issueState"
								:included="sections.state.included"
								:title="$t('issue.additional.state')"
								:content="sections.state.content"
								:helpType="helpType"
								@update:included="sections.state.included = $event"
								@update:content="sections.state.content = $event"
							>
								<template #description>
									<p class="text-muted small">
										{{ $t("issue.additional.stateDescription") }}<br />
										{{ $t("issue.additional.source") }}:
										<a :href="apiStateUrl" target="_blank">{{ apiStateUrl }}</a>
									</p>
								</template>
							</IssueAdditionalItem>
						</div>
					</div>

					<!-- Essential Section Actions -->
					<div class="d-flex justify-content-end gap-3 mb-5">
						<button type="submit" class="btn" :class="buttonClass">
							{{ buttonText }}
						</button>
					</div>
				</form>
			</main>
		</div>

		<!-- Issue Summary Modal -->
		<SummaryModal
			:help-type="helpType"
			:button-class="buttonClass"
			:issue-data="issueData"
			:sections="sections"
			@submitted="clearSessionStorage"
		/>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import TopHeader from "@/components/Top/Header.vue";
import MultiSelect from "@/components/Helper/MultiSelect.vue";
import IssueAdditionalItem from "@/components/Issue/AdditionalItem.vue";
import SummaryModal from "@/components/Issue/SummaryModal.vue";
import Modal from "bootstrap/js/dist/modal";
import api from "@/api";
import store from "@/store";
import { LOG_LEVELS, DEFAULT_LOG_LEVEL } from "@/utils/log";
import { formatJson } from "@/components/Issue/format";
import type { HelpType, IssueData } from "@/components/Issue/types";

// Keys that should be expanded (1-level expansion for arrays and objects)
const EXPAND_KEYS = [
	"battery",
	"forecast",
	"loadpoints",
	"pv",
	"vehicles",
	"statistics",
	"charger",
	"site",
	"meter",
	"vehicle",
];

type SectionType = "yamlConfig" | "uiConfig" | "logs" | "state";

interface SectionData {
	content: string;
	included: boolean;
}

export default defineComponent({
	name: "Issue",
	components: {
		TopHeader,
		MultiSelect,
		IssueAdditionalItem,
		SummaryModal,
	},
	data() {
		return {
			// Help type selection
			helpType: (sessionStorage.getItem("issue.helpType") as HelpType) || "discussion",

			// Essential form data
			issue: {
				title: sessionStorage.getItem("issue.title") || "",
				description: sessionStorage.getItem("issue.description") || "",
				steps: sessionStorage.getItem("issue.steps") || "",
			},

			// Section data with included flags
			sections: {
				yamlConfig: { content: "", included: true },
				uiConfig: { content: "", included: true },
				logs: { content: "", included: true },
				state: { content: "", included: false },
			} as Record<SectionType, SectionData>,

			// Log configuration
			logLevels: [...LOG_LEVELS],
			logLevel: DEFAULT_LOG_LEVEL,
			logCount: 25,
			logAreas: [] as string[],
			logAvailableAreas: [] as string[],
		};
	},
	computed: {
		versionString(): string {
			return `v${store.state.version || ""}`;
		},
		configPath(): string | undefined {
			return store.state.config;
		},
		databasePath(): string | undefined {
			return store.state.database;
		},
		issueData(): IssueData {
			return { ...this.issue, version: this.versionString };
		},
		logAreaOptions() {
			return this.logAvailableAreas.map((area) => ({ name: area, value: area }));
		},
		logAreasLabel() {
			if (this.logAreas.length === 0) {
				return this.$t("log.areas");
			}
			return this.logAreas.join(", ");
		},
		apiStateUrl(): string {
			const url = window.location.href.split("#")[0];
			return `${url}api/state`;
		},
		buttonClass(): string {
			return this.helpType === "discussion" ? "btn-success" : "btn-danger";
		},
		buttonText(): string {
			return this.$tt("issue.createButton");
		},
	},
	watch: {
		logLevel() {
			this.loadLogs();
		},
		logCount() {
			this.loadLogs();
		},
		logAreas() {
			this.loadLogs();
		},
		"issue.title"(newValue: string) {
			sessionStorage.setItem("issue.title", newValue);
		},
		"issue.description"(newValue: string) {
			sessionStorage.setItem("issue.description", newValue);
		},
		"issue.steps"(newValue: string) {
			sessionStorage.setItem("issue.steps", newValue);
		},
		helpType(newValue: string) {
			sessionStorage.setItem("issue.helpType", newValue);
		},
	},
	async mounted() {
		this.loadYamlConfig();
		this.loadUiConfig();
		this.loadState();
		this.loadLogs();
		this.updateAreas();
	},
	methods: {
		// Type-dependent translation helper
		$tt(key: string): string {
			const suffix = this.helpType === "discussion" ? "Discussion" : "Issue";
			return this.$t(`${key}${suffix}`);
		},

		async loadYamlConfig() {
			try {
				const response = await api.get("config/evcc.yaml", {
					responseType: "text",
					validateStatus: (code) => [200, 404].includes(code),
				});

				// Handle 404 silently when evcc.yaml doesn't exist
				if (response.status === 404) {
					this.sections.yamlConfig.content = "no yaml configuration";
					return;
				}

				// Remove empty lines from config
				this.sections.yamlConfig.content = response.data
					.split("\n")
					.filter((line: string) => line.trim() !== "")
					.join("\n");
			} catch (error: any) {
				console.error("Failed to fetch config:", error);
				this.sections.yamlConfig.content = "Failed to load configuration";
			}
		},

		async loadUiConfig() {
			try {
				const deviceEndpoints = [
					"config/loadpoints",
					"config/devices/charger",
					"config/devices/meter",
					"config/devices/vehicle",
				];

				const endpoints = [
					"config/site",
					...deviceEndpoints,
					"config/circuits",
					"config/eebus",
					"config/hems",
					"config/messaging",
					"config/modbusproxy",
					"config/tariffs",
				];

				const configs: any = {};

				for (const endpoint of endpoints) {
					try {
						const response = await api.get(endpoint);
						if (response.data && Object.keys(response.data).length > 0) {
							const key = endpoint.replace("config/", "").replace("devices/", "");
							let data = response.data;

							// Filter out entries without id property for device endpoints
							if (deviceEndpoints.includes(endpoint)) {
								if (Array.isArray(data)) {
									data = data.filter((e) => e.id);
								}
							}

							configs[key] = data;
						}
					} catch (error) {
						console.error(`Failed to fetch ${endpoint}:`, error);
					}
				}

				this.sections.uiConfig.content = formatJson(configs, EXPAND_KEYS);
			} catch (error) {
				console.error("Failed to fetch API config:", error);
				this.sections.uiConfig.content = "Failed to load API configuration";
			}
		},

		async loadLogs() {
			try {
				const params: any = {
					level: this.logLevel,
					count: this.logCount,
					area: this.logAreas.length ? this.logAreas : null,
				};

				const response = await api.get("/system/log", { params });
				const logs = response.data || [];
				this.sections.logs.content = logs
					.filter((entry: string) => entry && entry.trim())
					.map((entry: string) => entry.trim())
					.join("\n");
			} catch (error) {
				console.error("Failed to fetch logs:", error);
				this.sections.logs.content = "Failed to load logs";
			}
		},

		async updateAreas() {
			try {
				const response = await api.get("/system/log/areas");
				this.logAvailableAreas = response.data || [];
			} catch (error) {
				console.error("Failed to load log areas:", error);
			}
		},

		async loadState() {
			try {
				const response = await api.get("state");
				this.sections.state.content = formatJson(response.data, EXPAND_KEYS);
			} catch (error) {
				console.error("Failed to fetch state:", error);
				this.sections.state.content = "Failed to load system state";
			}
		},

		handleFormSubmit() {
			const modalElement = document.getElementById("issueSummaryModal") as HTMLElement;
			if (modalElement) {
				Modal.getOrCreateInstance(modalElement).show();
			}
		},

		clearSessionStorage() {
			sessionStorage.removeItem("issue.title");
			sessionStorage.removeItem("issue.description");
			sessionStorage.removeItem("issue.steps");
			sessionStorage.removeItem("issue.helpType");
		},
	},
});
</script>

<style scoped>
.log-count-input::-webkit-outer-spin-button,
.log-count-input::-webkit-inner-spin-button {
	margin-left: 0.5rem;
}

@media (min-width: 768px) {
	.log-lines-input {
		width: 170px;
	}
	.log-areas-select {
		width: 220px;
	}
}
</style>
