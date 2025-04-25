<template>
	<Teleport to="body">
		<div
			id="helpModal"
			class="modal fade text-dark"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">{{ $t("help.modalTitle") }}</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<p>
							{{ $t("help.primaryActions") }}
						</p>
						<div
							class="d-block d-sm-flex justify-content-between align-items-stretch mb-4"
						>
							<a
								:href="docsUrl"
								target="_blank"
								class="btn btn-outline-primary w-100 w-sm-auto flex-grow-1 mb-3 mb-sm-0 me-sm-3"
								type="button"
							>
								{{ $t("help.documentationButton") }}
							</a>
							<a
								href="https://github.com/evcc-io/evcc/discussions"
								target="_blank"
								class="btn btn-outline-primary w-100 w-sm-auto flex-grow-1"
								type="button"
							>
								{{ $t("help.discussionsButton") }}
							</a>
						</div>
						<hr class="mb-4" />
						<p>
							{{ $t("help.secondaryActions") }}
						</p>
						<div
							class="d-block d-sm-flex justify-content-between align-items-baseline mb-3"
						>
							<p class="flex-sm-grow-1 opacity-50 me-sm-3">
								{{ $t("help.logsDescription") }}
							</p>
							<router-link to="/log" class="btn btn-outline-primary text-nowrap">
								{{ $t("help.logsButton") }}
							</router-link>
						</div>
						<div
							class="d-block d-sm-flex justify-content-between align-items-baseline mb-3"
						>
							<p class="flex-sm-grow-1 opacity-50 me-sm-3">
								{{ $t("help.issueDescription") }}
							</p>
							<a
								href="https://github.com/evcc-io/evcc/issues"
								target="_blank"
								class="btn btn-outline-primary text-nowrap"
							>
								{{ $t("help.issueButton") }}
							</a>
						</div>

						<div
							class="d-block d-sm-flex justify-content-between align-items-baseline mb-3"
						>
							<p class="flex-sm-grow-1 opacity-50 me-sm-3">
								{{ $t("help.restartDescription") }}
							</p>
							<button
								class="btn btn-outline-danger text-nowrap"
								type="button"
								data-bs-dismiss="modal"
								@click="openConfirmRestartModal"
							>
								{{ $t("help.restartButton") }}
							</button>
						</div>

						<div
							class="d-block d-sm-flex justify-content-between align-items-baseline mb-3"
						>
							<p class="flex-sm-grow-1 opacity-50 me-sm-3">
								{{ $t("help.discussDescription") }}
							</p>
							<button
								class="btn btn-outline-primary text-nowrap"
								type="button"
								@click="openDiscussModal"
							>
								{{ $t("help.discussButton") }}
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
	<Teleport to="body">
		<div
			id="confirmRestartModal"
			class="modal fade text-dark"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5>{{ $t("help.restart.modalTitle") }}</h5>
					</div>
					<div class="modal-body">
						<p>{{ $t("help.restart.description") }}</p>
						<p>
							<small>
								{{ $t("help.restart.disclaimer") }}
							</small>
						</p>
					</div>
					<div class="modal-footer d-flex justify-content-between">
						<button
							type="button"
							class="btn btn-link text-muted"
							data-bs-dismiss="modal"
							@click="openHelpModal"
						>
							{{ $t("help.restart.cancel") }}
						</button>
						<button
							type="button"
							class="btn btn-danger"
							data-bs-dismiss="modal"
							@click="restartConfirmed"
						>
							{{ $t("help.restart.confirm") }}
						</button>
					</div>
				</div>
			</div>
		</div>
	</Teleport>

	<Teleport to="body">
		<div
			id="discussModal"
			class="modal fade text-dark"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-lg" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">{{ $t("help.discuss.modalTitle") }}</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<p>{{ $t("help.discuss.description") }}</p>
						<p>
							<small>
								{{ $t("help.discuss.disclaimer") }}
							</small>
						</p>

						<!-- Editierbarer Bereich für die Fehlerbeschreibung -->
						<textarea
							v-model="errorDescription"
							class="form-control mb-3"
							rows="5"
							placeholder="Fehlerbeschreibung hier eingeben..."
						></textarea>

						<!-- Bereich für generierte Daten mit Copy-Button -->
						<div class="generated-data-section">
							<label for="generatedData">{{
								$t("help.discuss.generatedDataLabel")
							}}</label>
							<textarea
								id="generatedData"
								v-model="generatedData"
								class="form-control mb-2"
								rows="10"
								readonly
								style="white-space: pre-wrap; word-wrap: break-word"
							></textarea>
							<button class="btn btn-outline-primary" @click="copyGeneratedData">
								{{ $t("help.discuss.copyButton") }}
							</button>
						</div>

						<!-- Bereich für Log-Daten unterhalb der generierten Daten hinzufügen -->
						<div class="log-data-section mt-4">
							<label for="logData">{{ $t("help.discuss.logDataLabel") }}</label>
							<textarea
								id="logData"
								v-model="logData"
								class="form-control mb-2"
								rows="10"
								readonly
								style="white-space: pre-wrap; word-wrap: break-word"
							></textarea>
						</div>

						<!-- Checkbox hinzufügen, um die Auswahl zu ermöglichen, ob Logs übergeben werden sollen -->
						<div class="form-check mt-3">
							<input
								id="includeLogsCheckbox"
								v-model="includeLogs"
								class="form-check-input"
								type="checkbox"
							/>
							<label class="form-check-label" for="includeLogsCheckbox">
								{{ $t("help.discuss.includeLogs") }}
							</label>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">
							{{ $t("help.discuss.cancel") }}
						</button>
						<a :href="discussUrl" target="_blank" class="btn btn-primary">
							{{ $t("help.discuss.submit") }}
						</a>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import { docsPrefix } from "../i18n.js";
import { performRestart } from "../restart.js";
import { isLoggedIn, openLoginModal } from "./Auth/auth";
import api from "../api";

export default {
	name: "HelpModal",

	props: {},
	data() {
		return {
			discussContent: "",
			discussUrl: "",
			errorDescription: "",
			generatedData: "",
			includeLogs: false, // Neue Variable für die Checkbox
			logData: "", // Neue Variable für die Log-Daten
		};
	},
	computed: {
		docsUrl() {
			return `${docsPrefix()}/`;
		},
	},
	watch: {
		errorDescription() {
			this.updateDiscussContent();
		},
		includeLogs() {
			this.updateDiscussContent();
		},
	},
	methods: {
		updateDiscussContent() {
			const base64GeneratedData = btoa(this.generatedData); // Base64-Encoding der generatedData
			this.discussContent = `<!-- Detaillierte Problembeschreibung bitte hier -->\n\n${this.errorDescription || "Keine Beschreibung angegeben."}\n\n${this.includeLogs ? `Logs:\n\n\`\`\`\n${this.logData}\n\`\`\`\n\n` : ""}`;
			this.discussUrl = `https://github.com/evcc-io/evcc/discussions/new?category=erste-hilfe&body=${encodeURIComponent(this.discussContent)}&data=${encodeURIComponent(base64GeneratedData)}`;
		},
		openHelpModal() {
			const modal = Modal.getOrCreateInstance(document.getElementById("helpModal"));
			modal.show();
		},
		openConfirmRestartModal() {
			const modal = Modal.getOrCreateInstance(document.getElementById("confirmRestartModal"));
			if (!isLoggedIn()) {
				openLoginModal(null, modal);
			} else {
				modal.show();
			}
			if (!isLoggedIn()) {
				openLoginModal(null, modal);
			} else {
				modal.show();
			}
		},
		async restartConfirmed() {
			await performRestart();
		},
		async openDiscussModal() {
			const modal = Modal.getOrCreateInstance(document.getElementById("discussModal"));

			// Fetch the configuration and logs
			try {
				const stateResponse = await api.get("/state");
				const stateData = stateResponse.data?.result || {};

				const logResponse = await api.get("/system/log", {
					params: { level: "DEBUG", count: 5 },
				});
				const logData = logResponse.data?.result || [];

				// Füge den Output von stateData in generatedData ein
				this.generatedData = JSON.stringify(stateData, null, 2); // Formatiere stateData als JSON und weise es generatedData zu

				// Überprüfe das Format von logData vor der Zuweisung
				if (Array.isArray(logData)) {
					// Verarbeite die Log-Daten direkt als Strings, da sie keine `message`-Eigenschaft haben
					const processedLogData = logData
						.map((log) => (typeof log === "string" ? log.trim() : ""))
						.filter((message) => message)
						.join("\n");
					this.logData = processedLogData;
				} else {
					this.logData = "Fehler beim Abrufen der Log-Daten.";
				}
				// Assign values to variables used in the template
				this.cfgError =
					logData.length > 0
						? logData.map((log) => log.message).join("\n")
						: "Keine Fehler gefunden.";

				// Überprüfe, ob die Logs korrekt in discussContent aufgenommen werden
				this.discussContent = `<!-- Detaillierte Problembeschreibung bitte hier -->\n\n${this.errorDescription || "Keine Beschreibung angegeben."}\n\n${this.includeLogs ? `Logs:\n\n\`\`\`\n${this.logData}\n\`\`\`\n\n` : ""}`;

				// Aktualisiere discussUrl, um sicherzustellen, dass discussContent korrekt übergeben wird
				this.discussUrl = `https://github.com/evcc-io/evcc/discussions/new?category=erste-hilfe&body=${encodeURIComponent(this.discussContent)}`;
			} catch (error) {
				console.error("Error fetching data for discussion:", error);
				this.cfgError = "Fehler beim Abrufen der Daten.";
				this.discussContent = "Error fetching configuration or logs.";
				this.discussUrl =
					"https://github.com/evcc-io/evcc/discussions/new?category=erste-hilfe";
			}

			modal.show();
		},
		copyGeneratedData() {
			navigator.clipboard
				.writeText(this.generatedData)
				.then(() => {
					alert("Generated data copied to clipboard!");
				})
				.catch((err) => {
					console.error("Failed to copy generated data:", err);
				});
		},
	},
};
</script>
