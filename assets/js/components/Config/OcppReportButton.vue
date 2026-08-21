<template>
	<div class="report-group d-flex align-items-center gap-3 ms-auto">
		<code v-if="rule" class="report-host text-truncate" :class="hostClass">{{ host }}</code>
		<button
			type="button"
			class="loadpoint-report btn d-flex align-items-center justify-content-center p-2 flex-shrink-0"
			:class="buttonClass"
			:title="title"
			:aria-label="title"
			data-bs-toggle="tooltip"
			@click="edit"
		>
			<OcppForwardStatus :status="status" />
		</button>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import OcppForwardStatus from "../MaterialIcon/OcppForwardStatus.vue";
import type { OcppReportRule } from "@/types/evcc";
import { openModal } from "@/configModal";

type ReportStatus = "unconfigured" | "configured" | "error";

export default defineComponent({
	name: "OcppReportButton",
	components: { OcppForwardStatus },
	props: {
		loadpointTitle: { type: String, required: true },
		rule: { type: Object as PropType<OcppReportRule>, default: undefined },
		error: { type: String, default: undefined },
	},
	computed: {
		status(): ReportStatus {
			if (!this.rule) return "unconfigured";
			return this.error ? "error" : "configured";
		},
		// hostname of the upstream URL, scheme and path stripped
		host(): string {
			if (!this.rule) return "";
			try {
				return new URL(this.rule.upstreamUrl).host;
			} catch {
				return this.rule.upstreamUrl || "";
			}
		},
		hostClass(): string {
			return this.status === "error" ? "text-danger" : "text-success";
		},
		buttonClass(): string {
			switch (this.status) {
				case "configured":
					return "text-success border border-success report-bg-success";
				case "error":
					return "text-danger border border-danger report-bg-error";
				default:
					return "report-muted border border-dashed";
			}
		},
		title(): string {
			switch (this.status) {
				case "configured":
					return this.$t("config.ocpp.forwardingConfigured");
				case "error":
					return this.$t("config.ocpp.forwardingError");
				default:
					return this.$t("config.ocpp.forwardingOff");
			}
		},
	},
	methods: {
		edit() {
			openModal("ocppreport", { loadpoint: this.loadpointTitle });
		},
	},
});
</script>

<style scoped>
.report-host {
	min-width: 0;
	max-width: 16rem;
	text-align: right;
	font-size: var(--bs-body-font-size);
}
.report-group {
	min-width: 0;
	flex-shrink: 100;
}
.report-muted {
	color: var(--bs-gray-light);
}
.border-dashed {
	border-style: dashed !important;
}
.report-bg-success {
	background-color: color-mix(in srgb, var(--evcc-primary) 10%, transparent);
}
.report-bg-error {
	background-color: color-mix(in srgb, var(--evcc-red) 10%, transparent);
}
</style>
