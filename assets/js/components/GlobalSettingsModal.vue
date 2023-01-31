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
			<div class="modal-dialog modal-dialog-centered" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">{{ $t("settings.title") }}</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<div class="container">
							<FormRow id="settingsDesign" :label="$t('settings.theme.label')">
								<SelectGroup
									id="settingsDesign"
									v-model="theme"
									class="w-100"
									:options="
										THEMES.map((value) => ({
											value,
											name: $t(`settings.theme.${value}`),
										}))
									"
								/>
							</FormRow>
							<FormRow id="settingsLanguage" :label="$t('settings.language.label')">
								<select
									id="settingsLanguage"
									v-model="language"
									class="form-select form-select-sm w-75"
								>
									<option value="">{{ $t("settings.language.auto") }}</option>
									<option
										v-for="option in languageOptions"
										:key="option"
										:value="option.value"
									>
										{{ option.name }}
									</option>
								</select>
							</FormRow>
							<FormRow id="settingsUnit" :label="$t('settings.unit.label')">
								<SelectGroup
									id="settingsUnit"
									v-model="unit"
									class="w-75"
									:options="
										UNITS.map((value) => ({
											value,
											name: $t(`settings.unit.${value}`),
										}))
									"
								/>
							</FormRow>
							<FormRow id="telemetryEnabled" :label="$t('settings.telemetry.label')">
								<TelemetrySettings :sponsor="sponsor" class="mt-1 mb-0" />
							</FormRow>
						</div>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import TelemetrySettings from "./TelemetrySettings.vue";
import FormRow from "./FormRow.vue";
import SelectGroup from "./SelectGroup.vue";
import { getLocalePreference, setLocalePreference, LOCALES, removeLocalePreference } from "../i18n";
import { getThemePreference, setThemePreference, THEMES } from "../theme";
import { getUnits, setUnits, UNITS } from "../units";

export default {
	name: "GlobalSettingsModal",
	components: { TelemetrySettings, FormRow, SelectGroup },
	props: {
		sponsor: String,
	},
	data: function () {
		return {
			theme: getThemePreference(),
			language: getLocalePreference() || "",
			unit: getUnits(),
			THEMES,
			UNITS,
		};
	},
	computed: {
		languageOptions: () => {
			const locales = Object.entries(LOCALES).map(([key, value]) => {
				return { value: key, name: value[1] };
			});
			// sort by name
			locales.sort((a, b) => (a.name < b.name ? -1 : 1));
			return locales;
		},
	},
	watch: {
		unit(value) {
			setUnits(value);
		},
		theme(value) {
			setThemePreference(value);
		},
		language(value) {
			const i18n = this.$root.$i18n;
			if (value) {
				setLocalePreference(i18n, value);
			} else {
				removeLocalePreference(i18n);
			}
		},
	},
};
</script>
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
}
</style>
