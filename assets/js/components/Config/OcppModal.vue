<template>
	<GenericModal
		id="ocppModal"
		config-modal-name="ocpp"
		:title="$t('config.ocpp.title')"
		data-testid="ocpp-modal"
	>
		<div class="container">
			<!-- OCPP URL -->
			<div class="mb-3">
				<Markdown :markdown="$t('config.ocpp.urlHelp', { url: ocppUrlWithStationId })" />
			</div>
			<FormRow id="ocppModalServerUrl" :label="$t('config.ocpp.url')">
				<input
					id="ocppModalServerUrl"
					type="text"
					class="form-control border"
					:value="ocppUrl"
					readonly
				/>
			</FormRow>

			<hr class="my-4" />

			<!-- No chargers message -->
			<div v-if="entries.length === 0" class="text-muted">
				{{ $t("config.ocpp.noChargers") }}
			</div>

			<!-- Stations -->
			<div v-else>
				<h6 class="mb-3">{{ $t("config.ocpp.stations") }}</h6>
				<p class="text-muted small mb-3">{{ $t("config.ocpp.stationsHelp") }}</p>

				<ul class="list-unstyled d-flex flex-column gap-3 mb-0">
					<li
						v-for="entry in entries"
						:key="entry.id"
						class="station-bar border rounded d-flex align-items-center gap-3 px-3 py-3"
						data-testid="ocpp-station"
					>
						<div class="text-truncate me-auto">
							<div v-if="entry.title" class="fw-bold fs-6 text-truncate">
								{{ entry.title }}
							</div>
							<code class="text-muted" :class="entry.title ? 'small' : 'fs-6'">{{
								entry.id
							}}</code>
						</div>
						<StatusIndicator :variant="statusVariant(entry.status)">
							{{ $t(`config.ocpp.status.${entry.status}`) }}
						</StatusIndicator>
						<button
							type="button"
							class="btn d-flex align-items-center justify-content-center p-2 flex-shrink-0"
							:class="forwardClass(entry)"
							:title="forwardTitle(entry)"
							:aria-label="forwardTitle(entry)"
							data-bs-toggle="tooltip"
							@click="editForwarder(entry.id)"
						>
							<OcppForwardStatus :status="forwardStatus(entry)" />
						</button>
					</li>
				</ul>
			</div>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import FormRow from "./FormRow.vue";
import Markdown from "./Markdown.vue";
import OcppForwardStatus from "../MaterialIcon/OcppForwardStatus.vue";
import StatusIndicator from "./StatusIndicator.vue";
import type {
	Ocpp,
	OcppForwarderRule,
	OcppForwarderSession,
	OcppStationStatus,
} from "@/types/evcc";
import { OCPP_STATION_STATUS } from "@/types/evcc";
import { getOcppUrl, getOcppUrlWithStationId } from "@/utils/ocpp";
import { openModal } from "@/configModal";
import store from "@/store";

type StationEntry = {
	id: string;
	status: OcppStationStatus["status"];
	title?: string;
	rule?: OcppForwarderRule;
	error?: string;
};

export default defineComponent({
	name: "OcppModal",
	components: {
		GenericModal,
		FormRow,
		Markdown,
		OcppForwardStatus,
		StatusIndicator,
	},
	props: {
		ocpp: {
			type: Object as PropType<Ocpp>,
			default: () => ({ config: { port: 0 }, status: { stations: [] } }),
		},
		stationTitles: {
			type: Object as PropType<Record<string, string>>,
			default: () => ({}),
		},
	},
	computed: {
		ocppUrl(): string {
			return getOcppUrl(this.ocpp);
		},
		ocppUrlWithStationId(): string {
			return getOcppUrlWithStationId(this.ocpp);
		},
		rules(): OcppForwarderRule[] {
			return store.state?.ocppforwarder?.config || [];
		},
		sessions(): OcppForwarderSession[] {
			return store.state?.ocppforwarder?.status || [];
		},
		// merge of published stations and configured forwarder rules, keyed by id.
		// a rule without a matching station is shown as "unknown".
		entries(): StationEntry[] {
			const byId = new Map<string, StationEntry>();
			for (const station of this.ocpp.status.stations) {
				byId.set(station.id, {
					id: station.id,
					status: station.status,
					title: this.stationTitles[station.id],
				});
			}
			for (const rule of this.rules) {
				if (rule.stationId === "*") continue;
				const entry = byId.get(rule.stationId) || {
					id: rule.stationId,
					status: OCPP_STATION_STATUS.UNKNOWN,
					title: this.stationTitles[rule.stationId],
				};
				entry.rule = rule;
				entry.error = this.sessions.find((s) => s.chargerId === rule.stationId)?.error;
				byId.set(rule.stationId, entry);
			}
			return [...byId.values()];
		},
	},
	methods: {
		statusVariant(status: string): "success" | "warning" | "muted" {
			switch (status) {
				case "connected":
					return "success";
				case "configured":
					return "warning";
				default:
					return "muted";
			}
		},
		forwardStatus(entry: StationEntry): "unconfigured" | "configured" | "error" {
			if (!entry.rule) return "unconfigured";
			return entry.error ? "error" : "configured";
		},
		forwardClass(entry: StationEntry): string {
			switch (this.forwardStatus(entry)) {
				case "configured":
					return "text-success border border-success forward-bg-success";
				case "error":
					return "text-danger border border-danger forward-bg-error";
				default:
					return "forward-muted border";
			}
		},
		forwardTitle(entry: StationEntry): string {
			switch (this.forwardStatus(entry)) {
				case "configured":
					return this.$t("config.ocpp.forwardingConfigured");
				case "error":
					return this.$t("config.ocpp.forwardingError");
				default:
					return this.$t("config.ocpp.forwardingOff");
			}
		},
		editForwarder(stationId: string) {
			openModal("ocppforwarder", { station: stationId });
		},
	},
});
</script>

<style scoped>
.container {
	padding-right: 0;
	padding-left: 0;
}

.station-bar {
	--bs-border-color: var(--bs-gray-light);
}

/* forward button tints, scoped so global subtle colors stay untouched */
.forward-muted {
	color: var(--bs-gray-light);
}
.forward-bg-success {
	background-color: color-mix(in srgb, var(--evcc-primary) 10%, transparent);
}
.forward-bg-error {
	background-color: color-mix(in srgb, var(--evcc-red) 10%, transparent);
}
</style>
