<template>
	<div class="container mx-0 px-0">
		<FormRow id="settingsDesign" :label="$t('settings.theme.label')">
			<SelectGroup
				id="settingsDesign"
				v-model="theme"
				class="w-100"
				transparent
				:options="
					THEMES.map((value) => ({
						value,
						name: $t(`settings.theme.${value}`),
					}))
				"
				equal-width
			/>
		</FormRow>
		<FormRow id="settingsLanguage" :label="$t('settings.language.label')">
			<select
				id="settingsLanguage"
				v-model="language"
				class="form-select form-select-sm w-75"
			>
				<option value="">{{ $t("settings.language.auto") }}</option>
				<option v-for="option in languageOptions" :key="option.value" :value="option.value">
					{{ option.name }}
				</option>
			</select>
		</FormRow>
		<FormRow id="settingsUnit" :label="$t('settings.unit.label')">
			<SelectGroup
				id="settingsUnit"
				v-model="unit"
				class="w-75"
				transparent
				:options="
					UNITS.map((value) => ({
						value,
						name: $t(`settings.unit.${value}`),
					}))
				"
				equal-width
			/>
		</FormRow>
		<FormRow id="settingsTimeFormat" :label="$t('settings.time.label')">
			<SelectGroup
				id="settingsTimeFormat"
				v-model="timeFormat"
				class="w-75"
				transparent
				:options="
					TIME_FORMATS.map((value) => ({
						value,
						name: $t(`settings.time.${value}h`),
					}))
				"
				equal-width
			/>
		</FormRow>
		<FormRow id="telemetryEnabled" :label="$t('settings.telemetry.label')">
			<TelemetrySettings :sponsorActive="sponsor && !!sponsor.name" class="mt-1 mb-0" />
		</FormRow>
		<FormRow id="hiddenFeaturesEnabled" :label="`${$t('settings.hiddenFeatures.label')} 🧪`">
			<div class="form-check form-switch my-1">
				<input
					id="hiddenFeaturesEnabled"
					v-model="hiddenFeatures"
					class="form-check-input"
					type="checkbox"
					role="switch"
				/>
				<div class="form-check-label">
					<label for="hiddenFeaturesEnabled">
						{{ $t("settings.hiddenFeatures.value") }}
					</label>
				</div>
			</div>
		</FormRow>
		<FormRow v-if="fullscreenAvailable" :label="$t('settings.fullscreen.label')">
			<button
				v-if="fullscreenActive"
				class="btn btn-sm btn-outline-secondary"
				@click="exitFullscreen"
			>
				{{ $t("settings.fullscreen.exit") }}
			</button>
			<button v-else class="btn btn-sm btn-outline-secondary" @click="enterFullscreen">
				{{ $t("settings.fullscreen.enter") }}
			</button>
		</FormRow>
	</div>
</template>

<script lang="ts">
import TelemetrySettings from "../TelemetrySettings.vue";
import FormRow from "../Helper/FormRow.vue";
import SelectGroup from "../Helper/SelectGroup.vue";
import { getLocalePreference, setLocalePreference, LOCALES, removeLocalePreference } from "@/i18n";
import { getThemePreference, setThemePreference, THEMES } from "@/theme";
import { getUnits, setUnits, UNITS, is12hFormat, set12hFormat } from "@/units";
import { getHiddenFeatures, setHiddenFeatures } from "@/featureflags";
import { isApp } from "@/utils/native";
import { defineComponent, type PropType } from "vue";
import type { Sponsor } from "@/types/evcc";

const TIME_12H = "12";
const TIME_24H = "24";

export default defineComponent({
	name: "UserInterfaceSettings",
	components: { TelemetrySettings, FormRow, SelectGroup },
	props: {
		sponsor: Object as PropType<Sponsor>,
	},
	data() {
		return {
			theme: getThemePreference(),
			language: getLocalePreference() || "",
			unit: getUnits(),
			timeFormat: is12hFormat() ? TIME_12H : TIME_24H,
			hiddenFeatures: getHiddenFeatures(),
			fullscreenActive: false,
			THEMES,
			UNITS,
			TIME_FORMATS: [TIME_24H, TIME_12H],
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
		fullscreenAvailable: () => {
			const isSupported = document.fullscreenEnabled;
			const isPwa =
				(navigator as any).standalone ||
				window.matchMedia("(display-mode: standalone)").matches;
			return isSupported && !isPwa && !isApp();
		},
	},
	watch: {
		unit(value) {
			setUnits(value);
		},
		timeFormat(value) {
			set12hFormat(value === TIME_12H);
		},
		theme(value) {
			setThemePreference(value);
		},
		hiddenFeatures(value) {
			setHiddenFeatures(value);
		},
		language(value) {
			const i18n = this.$root?.$i18n;
			if (value) {
				setLocalePreference(i18n, value);
			} else {
				removeLocalePreference(i18n);
			}
		},
	},
	mounted() {
		document.addEventListener("fullscreenchange", this.fullscreenChange);
	},
	unmounted() {
		document.removeEventListener("fullscreenchange", this.fullscreenChange);
	},
	methods: {
		enterFullscreen() {
			document.documentElement.requestFullscreen();
		},
		exitFullscreen() {
			document.exitFullscreen();
		},
		fullscreenChange() {
			this.fullscreenActive = !!document.fullscreenElement;
		},
	},
});
</script>
