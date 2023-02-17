<!-- eslint-disable vue/no-v-html -->
<template>
	<div class="form-check form-switch my-3">
		<input
			id="telemetryEnabled"
			v-model="enabled"
			class="form-check-input"
			type="checkbox"
			role="switch"
			:disabled="!sponsor"
			@change="change"
		/>
		<div class="form-check-label">
			<label for="telemetryEnabled">
				{{ $t("footer.telemetry.optIn") }}
				<i18n-t v-if="sponsor" tag="span" keypath="footer.telemetry.optInMoreDetails">
					<a
						href="https://docs.evcc.io/docs/guides/setup/#telemetry--community-daten"
						target="_blank"
					>
						{{ $t("footer.telemetry.optInMoreDetailsLink") }}
					</a>
				</i18n-t>
				<span v-else>{{ $t("footer.telemetry.optInSponsorship") }}</span>
			</label>
			<div v-if="error" class="errorMessage my-1 text-danger" v-html="error" />
		</div>
	</div>
</template>

<script>
import api from "../api";
import settings from "../settings";

function parseMarkdown(markdownText) {
	const htmlText = markdownText
		.replace(/\*\*(.*)\*\*/gim, "<b>$1</b>")
		.replace(/\*(.*)\*/gim, "<i>$1</i>")
		.replace(/`(.*)`/gim, "<pre>$1</pre>");
	return htmlText.trim();
}

export default {
	name: "TelemetrySettings",
	props: { sponsor: String },
	data() {
		return {
			error: null,
		};
	},
	computed: {
		enabled() {
			return settings.telemetry;
		},
	},
	async mounted() {
		await this.update();
	},
	methods: {
		async change(e) {
			try {
				this.error = null;
				const response = await api.post(`settings/telemetry/${e.target.checked}`);
				settings.telemetry = response.data.result;
			} catch (err) {
				if (err.response) {
					this.error = parseMarkdown("**Error:** " + err.response.data.error);
					settings.telemetry = false;
				}
			}
		},
		async update() {
			if (settings.telemetry !== null) {
				return;
			}
			try {
				const response = await api.get("settings/telemetry");
				settings.telemetry = response.data.result;
			} catch (err) {
				console.error(err);
			}
		},
	},
};
</script>
<style scoped>
.form-check {
	min-height: inherit !important;
}
.form-check-label {
	max-width: 100%;
}
.errorMessage :deep(pre) {
	text-overflow: ellipsis;
	font-size: 1em;
}
</style>
