<template>
	<div class="forward-group d-flex align-items-center gap-3 ms-auto">
		<code v-if="rule" class="station-host text-truncate" :class="hostClass">{{ host }}</code>
		<button
			type="button"
			class="station-forward btn d-flex align-items-center justify-content-center p-2 flex-shrink-0"
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
import type { OcppForwarderRule } from "@/types/evcc";
import { openModal } from "@/configModal";

type ForwardStatus = "unconfigured" | "configured" | "error";

export default defineComponent({
	name: "OcppForwarderButton",
	components: { OcppForwardStatus },
	props: {
		stationId: { type: String, required: true },
		rule: { type: Object as PropType<OcppForwarderRule>, default: undefined },
		error: { type: String, default: undefined },
	},
	computed: {
		status(): ForwardStatus {
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
		// host color tracks the button: green when configured, red on error
		hostClass(): string {
			return this.status === "error" ? "text-danger" : "text-success";
		},
		buttonClass(): string {
			switch (this.status) {
				case "configured":
					return "text-success border border-success forward-bg-success";
				case "error":
					return "text-danger border border-danger forward-bg-error";
				default:
					return "forward-muted border border-dashed";
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
			openModal("ocppforwarder", { station: this.stationId });
		},
	},
});
</script>

<style scoped>
.station-host {
	min-width: 0;
	max-width: 16rem;
	text-align: right;
	font-size: var(--bs-body-font-size);
}
/* single line: the host (with its button) gives way and truncates before the identity */
.forward-group {
	min-width: 0;
	flex-shrink: 100;
}

/* button tints, scoped so global subtle colors stay untouched */
.forward-muted {
	color: var(--bs-gray-light);
}
.border-dashed {
	border-style: dashed !important;
}
.forward-bg-success {
	background-color: color-mix(in srgb, var(--evcc-primary) 10%, transparent);
}
.forward-bg-error {
	background-color: color-mix(in srgb, var(--evcc-red) 10%, transparent);
}
</style>
