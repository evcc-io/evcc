<!-- eslint-disable vue/no-v-html -->
<template>
	<div class="form-check form-switch my-3">
		<input
			id="telemetryEnabled"
			:checked="telemetry"
			class="form-check-input"
			type="checkbox"
			role="switch"
			:disabled="!sponsorActive"
			@change="change"
		/>
		<div class="form-check-label">
			<label for="telemetryEnabled">
				{{ $t("footer.telemetry.optIn") }}
				<i18n-t
					v-if="sponsorActive"
					tag="span"
					keypath="footer.telemetry.optInMoreDetails"
					scope="global"
				>
					<a :href="docsLink" target="_blank">
						{{ $t("footer.telemetry.optInMoreDetailsLink") }}
					</a>
				</i18n-t>
				<span v-else>{{ $t("footer.telemetry.optInSponsorship") }}</span>
			</label>
			<div v-if="error" class="errorMessage my-1 text-danger" v-html="error" />
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import api from "../api";
import { docsPrefix } from "../i18n";
import type { AxiosError } from "axios";

function parseMarkdown(markdownText: string) {
	const htmlText = markdownText
		.replace(/\*\*(.*)\*\*/gim, "<b>$1</b>")
		.replace(/\*(.*)\*/gim, "<i>$1</i>")
		.replace(/`(.*)`/gim, "<pre>$1</pre>");
	return htmlText.trim();
}

export default defineComponent({
	name: "TelemetrySettings",
	props: { sponsorActive: Boolean, telemetry: Boolean },
	data() {
		return {
			error: null as string | null,
		};
	},
	computed: {
		docsLink() {
			return `${docsPrefix()}/docs/faq#telemetry`;
		},
	},
	methods: {
		async change(e: Event) {
			try {
				this.error = null;
				await api.post(`settings/telemetry/${(e.target as HTMLInputElement).checked}`);
			} catch (err) {
				const e = err as AxiosError<{ error: string }>;
				if (e.response) {
					this.error = parseMarkdown("**Error:** " + e.response.data.error);
				}
			}
		},
	},
});
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
