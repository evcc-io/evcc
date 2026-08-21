<template>
	<div class="report-group d-flex align-items-center ms-auto">
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

type ReportStatus = "unconfigured" | "pending" | "connected" | "error";

export default defineComponent({
	name: "OcppReportButton",
	components: { OcppForwardStatus },
	props: {
		loadpointTitle: { type: String, required: true },
		rule: { type: Object as PropType<OcppReportRule>, default: undefined },
		connected: { type: Boolean, default: false },
		error: { type: String, default: undefined },
	},
	computed: {
		status(): ReportStatus {
			if (!this.rule) return "unconfigured";
			if (this.error) return "error";
			return this.connected ? "connected" : "pending";
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
		buttonClass(): string {
			switch (this.status) {
				case "connected":
					return "text-success border border-success report-bg-success";
				case "pending":
					return "text-warning border border-warning report-bg-pending";
				case "error":
					return "text-danger border border-danger report-bg-error";
				default:
					return "report-muted border border-dashed";
			}
		},
		title(): string {
			const label = (() => {
				switch (this.status) {
					case "connected":
						return this.$t("config.ocppreport.statusConnected");
					case "pending":
						return this.$t("config.ocppreport.statusPending");
					case "error":
						return this.$t("config.ocppreport.statusError");
					default:
						return this.$t("config.ocppreport.statusUnconfigured");
				}
			})();
			return this.host ? `${label} (${this.host})` : label;
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
.report-bg-pending {
	background-color: color-mix(in srgb, var(--evcc-orange) 10%, transparent);
}
.report-bg-error {
	background-color: color-mix(in srgb, var(--evcc-red) 10%, transparent);
}
</style>
