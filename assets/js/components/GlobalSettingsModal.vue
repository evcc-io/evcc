<template>
	<Teleport to="body">
		<div
			id="globalSettingsModal"
			class="modal fade text-dark"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered modal-dialog-scrollable" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">Einstellungen</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<div class="container">
							<div class="mb-3 row">
								<label
									for="settingsDesign"
									class="col-sm-4 col-form-label pt-0 pt-sm-1"
								>
									{{ $t("settings.theme.label") }}
								</label>
								<div class="col-sm-8 pe-0">
									<select
										id="settingsDesign"
										v-model="theme"
										class="form-select form-select-sm mb-2 w-50"
									>
										<option
											v-for="option in ['auto', 'light', 'dark']"
											:key="option"
											:value="option"
										>
											{{ $t(`settings.theme.${option}`) }}
										</option>
									</select>
								</div>
							</div>

							<div class="mb-3 row">
								<label
									for="settingsDesign"
									class="col-sm-4 col-form-label pt-0 pt-sm-1"
								>
									{{ $t("settings.language.label") }}
								</label>
								<div class="col-sm-8 pe-0">
									<select
										id="settingsDesign"
										v-model="language"
										class="form-select form-select-sm mb-2 w-75"
									>
										<option
											v-for="option in languageOptions"
											:key="option"
											:value="option.value"
										>
											{{ option.name }}
										</option>
									</select>
								</div>
							</div>

							<div class="mb-3 row">
								<label
									for="settingsDesign"
									class="col-sm-4 col-form-label pt-0 pt-sm-1"
								>
									Telemetry
								</label>
								<div class="col-sm-8 pe-0">
									<TelemetrySettings class="mt-1" />
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import TelemetrySettings from "./TelemetrySettings.vue";
import { getLocalePreference, LOCALES } from "../i18n";
import { getThemePreference, setThemePreference, THEMES } from "../theme";

export default {
	name: "GlobalSettingsModal",
	components: { TelemetrySettings },
	data: function () {
		return {
			theme: getThemePreference() || "auto",
			language: getLocalePreference() || "auto",
		};
	},
	computed: {
		languageOptions: () => {
			const result = [{ value: "auto", name: "Automatisch" }];
			const locales = Object.entries(LOCALES);
			// sort by name
			locales.sort((a, b) => (a[1] < b[1] ? -1 : 1));
			locales.forEach(([key, value]) => {
				result.push({ value: key, name: value });
			});
			return result;
		},
	},

	methods: {
		toggleTheme: function () {
			const currentIndex = THEMES.indexOf(this.theme);
			const nextIndex = currentIndex < THEMES.length - 1 ? currentIndex + 1 : 0;
			this.theme = THEMES[nextIndex];
			setThemePreference(this.theme);
		},
	},
};
</script>
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
}

.container h4:first-child {
	margin-top: 0 !important;
}
</style>
