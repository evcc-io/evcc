<template>
	<div class="form-check form-switch my-3">
		<input
			id="telemetryEnabled"
			:checked="enabled"
			class="form-check-input"
			type="checkbox"
			role="switch"
			:disabled="!sponsor"
			@change="change"
		/>
		<label class="form-check-label" for="telemetryEnabled">
			{{ $t("footer.telemetry.optIn") }}
			<i18n-t v-if="sponsor" tag="span" keypath="footer.telemetry.optInMoreDetails">
				<a href="https://docs.evcc.io/docs/reference/configuration/telemetry/">
					{{ $t("footer.telemetry.optInMoreDetailsLink") }}
				</a>
				<!-- TODO: create a more user friendy explaination site -->
			</i18n-t>
			<span v-else>{{ $t("footer.telemetry.optInSponsorship") }}</span>
		</label>
	</div>
</template>

<script>
import api from "../api";

export default {
	name: "TelemetrySettings",
	props: { sponsor: String },
	data() {
		return {
			enabled: null,
		};
	},
	async mounted() {
		await this.update();
	},
	methods: {
		async change(e) {
			const response = await api.post(`settings/telemetry/${e.target.checked}`);
			this.enabled = response.data.result;
		},
		async update() {
			const response = await api.get("settings/telemetry");
			this.enabled = response.data.result;
		},
	},
};
</script>
<style scoped>
.form-check {
	min-height: inherit !important;
}
</style>
