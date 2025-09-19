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
									<small class="text-muted">{{
										$t("issue.helpType.discussionDescription")
									}}</small>
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
									<small class="text-muted">{{
										$t("issue.helpType.issueDescription")
									}}</small>
								</label>
							</div>
						</div>
					</div>
				</div>

				<form v-if="helpType" @submit.prevent="handleFormSubmit">
					<!-- Essential Form Section -->
					<div class="d-flex justify-content-between align-items-center mb-4">
						<h4>
							{{
								helpType === "discussion"
									? $t("issue.subTitle")
									: $t("issue.subTitleIssue")
							}}
						</h4>
					</div>

					<!-- Two Column Layout -->
					<div class="row mb-5 g-5">
						<!-- Left Column: Form Fields -->
						<div class="col-12 col-lg-6">
							<div class="mb-4">
								<label for="issueTitle" class="form-label"
									>{{ $t("issue.issueTitle") }} *</label
								>
								<input
									id="issueTitle"
									v-model="issue.title"
									type="text"
									class="form-control"
									:placeholder="$t('issue.issueTitlePlaceholder')"
									required
								/>
							</div>
							<div class="mb-4">
								<label for="issueDescription" class="form-label"
									>{{ $t("issue.issueDescription") }} *</label
								>
								<textarea
									id="issueDescription"
									v-model="issue.description"
									class="form-control"
									rows="6"
									:placeholder="$t('issue.issueDescriptionPlaceholder')"
									required
								></textarea>
							</div>
							<div class="mb-4">
								<label for="stepsToReproduce" class="form-label">
									{{
										helpType === "discussion"
											? $t("issue.additionalContext")
											: $t("issue.stepsToReproduce")
									}}
									*
								</label>
								<textarea
									id="stepsToReproduce"
									v-model="issue.steps"
									class="form-control"
									rows="6"
									:placeholder="
										helpType === 'discussion'
											? $t('issue.additionalContextPlaceholder')
											: $t('issue.stepsToReproducePlaceholder')
									"
									required
								></textarea>
							</div>
							<div class="mb-4">
								<label for="version" class="form-label">{{
									$t("issue.version")
								}}</label>
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
										<span v-if="configPath"
											>{{ $t("issue.additional.source") }}:
											<code>{{ configPath }}</code></span
										>
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
										<span v-if="databasePath"
											>{{ $t("issue.additional.source") }}:
											<code>{{ databasePath }}</code></span
										>
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
												<span class="input-group-text">{{
													$t("issue.additional.lines")
												}}</span>
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
		<GenericModal
			id="issueSummaryModal"
			:title="$t('issue.summary.title')"
			size="lg"
			:autofocus="false"
		>
			<!-- Instructions -->
			<div class="alert alert-secondary mb-4">
				<strong>{{ $t("issue.summary.instructions") }}</strong>
			</div>

			<!-- Step 1: Create Basic Issue -->
			<div class="mb-4">
				<h6 class="mb-2">
					{{
						helpType === "discussion"
							? $t("issue.summary.stepOneDiscussion")
							: $t("issue.summary.stepOneIssue")
					}}
				</h6>
				<p class="text-muted small mb-3">{{ $t("issue.summary.step1Description") }}</p>
				<div class="d-flex justify-content-start">
					<a
						:href="githubUrl"
						target="_blank"
						class="btn"
						:class="buttonClass"
						@click="clearSessionStorage"
					>
						{{
							helpType === "discussion"
								? $t("issue.summary.confirmationButtonDiscussion")
								: $t("issue.summary.confirmationButtonIssue")
						}}
					</a>
				</div>
			</div>

			<hr class="my-4" />

			<!-- Step 2: Copy Additional Information -->
			<div class="mb-4">
				<h6 class="mb-2">{{ $t("issue.summary.stepTwo") }}</h6>
				<p class="text-muted small mb-3">{{ $t("issue.summary.step2Description") }}</p>
				<div class="d-flex justify-content-start mb-4">
					<CopyButton :content="summaryDetails" :targetElement="$refs['summaryTextarea']">
						<template #default="{ copy, copied, copying }">
							<button
								type="button"
								class="btn"
								:class="
									helpType === 'discussion'
										? 'btn-outline-success'
										: 'btn-outline-danger'
								"
								:disabled="copying"
								@click="copy"
							>
								{{
									copied
										? $t("issue.summary.copied")
										: $t("issue.summary.copyButton")
								}}
							</button>
						</template>
					</CopyButton>
				</div>
				<div class="mb-2">
					<textarea
						ref="summaryTextarea"
						:value="summaryDetails"
						class="form-control font-monospace border-secondary textarea--tiny"
						:rows="summaryDetailsRows"
						readonly
						style="white-space: pre; overflow-wrap: normal"
						:placeholder="$t('issue.additional.combinedPlaceholder')"
					></textarea>
				</div>
			</div>
		</GenericModal>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import TopHeader from "@/components/Top/Header.vue";
import MultiSelect from "@/components/Helper/MultiSelect.vue";
import IssueAdditionalItem from "@/components/Issue/AdditionalItem.vue";
import GenericModal from "@/components/Helper/GenericModal.vue";
import CopyButton from "@/components/Helper/CopyButton.vue";
import Modal from "bootstrap/js/dist/modal";
import api from "@/api";
import store from "@/store";
import { LOG_LEVELS, DEFAULT_LOG_LEVEL } from "@/utils/log";

// Type definitions
export type HelpType = "discussion" | "issue";

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
		GenericModal,
		CopyButton,
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
		summaryDetails(): string {
			let output = "";

			if (this.sections.yamlConfig.included) {
				const compactConfig = this.sections.yamlConfig.content
					.split("\n")
					.filter((line: string) => line.trim() !== "")
					.join("\n");
				output += `## Configuration (evcc.yaml)\n\n\`\`\`yaml\n${compactConfig}\n\`\`\`\n\n`;
			}

			if (this.sections.uiConfig.included) {
				output += `## Configuration (UI)\n\n\`\`\`json5\n${this.sections.uiConfig.content}\n\`\`\`\n\n`;
			}

			if (this.sections.state.included) {
				output += `## System State\n\n\`\`\`json5\n${this.sections.state.content}\n\`\`\`\n\n`;
			}

			if (this.sections.logs.included) {
				output += `## Logs\n\n\`\`\`\n${this.sections.logs.content}\n\`\`\`\n\n`;
			}

			return output.trim() || "No additional details selected";
		},
		summaryDetailsRows(): number {
			const content = this.summaryDetails;
			const lines = content.split("\n").length;
			return Math.max(26, lines); // Min 3 rows, exact line count
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
		githubUrl(): string {
			const encodedTitle = encodeURIComponent(this.issue.title);
			const isDiscussion = this.helpType === "discussion";

			const mainSection = isDiscussion ? "## Description" : "## Issue Description";
			const stepsSection = isDiscussion ? "## Additional Context" : "## Steps to Reproduce";

			const body = `${mainSection}

${this.issue.description}

${
	this.issue.steps
		? `${stepsSection}

${this.issue.steps}

`
		: ""
}⚠️  RETURN TO EVCC TAB → COPY STEP 2 → PASTE HERE

## Version

${this.versionString}`;

			const encodedBody = encodeURIComponent(body);
			const baseUrl = isDiscussion
				? "https://github.com/evcc-io/evcc/discussions/new?category=need-help"
				: "https://github.com/evcc-io/evcc/issues/new";

			return `${baseUrl}&title=${encodedTitle}&body=${encodedBody}`;
		},

		buttonClass(): string {
			return this.helpType === "discussion" ? "btn-success" : "btn-danger";
		},

		buttonText(): string {
			return this.helpType === "discussion"
				? this.$t("issue.createDiscussionButton")
				: this.$t("issue.createIssueButton");
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
		formatJson(obj: any): string {
			if (!obj || typeof obj !== "object") {
				return JSON.stringify(obj, null, 2);
			}

			const lines: string[] = [];

			for (const [key, value] of Object.entries(obj)) {
				let valueStr: string;

				// Check if this key should be expanded (only if not empty)
				if (
					EXPAND_KEYS.includes(key) &&
					(Array.isArray(value) || (typeof value === "object" && value !== null))
				) {
					if (Array.isArray(value)) {
						if (value.length === 0) {
							// Keep empty arrays compact
							valueStr = "[]";
						} else {
							const arrayItems = value.map((item) => {
								const itemStr = JSON.stringify(item);
								return `    ${itemStr.replace(/\\n/g, "\n")}`;
							});
							valueStr = `[\n${arrayItems.join(",\n")}\n  ]`;
						}
					} else {
						// Object expansion
						const objEntries = Object.entries(value);
						if (objEntries.length === 0) {
							// Keep empty objects compact
							valueStr = "{}";
						} else {
							const objItems = objEntries.map(([k, v]) => {
								const itemStr = JSON.stringify(v);
								return `    ${JSON.stringify(k)}: ${itemStr.replace(/\\n/g, "\n")}`;
							});
							valueStr = `{\n${objItems.join(",\n")}\n  }`;
						}
					}
				} else {
					// Single line for everything else
					valueStr = JSON.stringify(value).replace(/\\n/g, "\n");
				}

				lines.push(`  ${JSON.stringify(key)}: ${valueStr}`);
			}

			return `{\n${lines.join(",\n")}\n}`;
		},

		async loadYamlConfig() {
			try {
				const response = await api.get("config/evcc.yaml", {
					responseType: "text",
				});
				// Remove empty lines from config
				this.sections.yamlConfig.content = response.data
					.split("\n")
					.filter((line: string) => line.trim() !== "")
					.join("\n");
			} catch (error) {
				console.error("Failed to fetch config:", error);
				this.sections.yamlConfig.content = "Failed to load configuration";
			}
		},

		async loadUiConfig() {
			try {
				const endpoints = [
					"config/site",
					"config/loadpoints",
					"config/devices/charger",
					"config/devices/meter",
					"config/devices/vehicle",
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
							configs[key] = response.data;
						}
					} catch (error) {
						console.error(`Failed to fetch ${endpoint}:`, error);
					}
				}

				this.sections.uiConfig.content = this.formatJson(configs);
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

				if (Array.isArray(response.data)) {
					const filteredLogs = response.data
						.filter((entry) => entry && entry.trim())
						.map((entry) => entry.trim());

					if (filteredLogs.length === 0) {
						this.sections.logs.content = "";
					} else {
						this.sections.logs.content = filteredLogs.join("\n");
					}
				} else {
					this.sections.logs.content = String(response.data);
				}
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
				this.sections.state.content = this.formatJson(response.data);
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
