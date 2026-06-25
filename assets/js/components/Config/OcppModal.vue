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
						<StatusIndicator
							class="flex-shrink-0"
							:variant="statusVariant(entry.status)"
							:tooltip="$t(`config.ocpp.status.${entry.status}`)"
						/>
						<div class="station-identity text-truncate">
							<div v-if="entry.title" class="fw-bold fs-6 text-truncate">
								{{ entry.title }}
							</div>
							<code class="text-muted" :class="entry.title ? 'small' : 'fs-6'">{{
								entry.id
							}}</code>
						</div>
						<OcppForwarderButton
							:station-id="entry.id"
							:rule="entry.rule"
							:error="entry.error"
						/>
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
import OcppForwarderButton from "./OcppForwarderButton.vue";
import StatusIndicator from "./StatusIndicator.vue";
import type {
	Ocpp,
	OcppForwarderRule,
	OcppForwarderSession,
	OcppStationStatus,
} from "@/types/evcc";
import { OCPP_STATION_STATUS } from "@/types/evcc";
import { getOcppUrl, getOcppUrlWithStationId } from "@/utils/ocpp";
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
		OcppForwarderButton,
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
			const published = new Set<string>();
			for (const station of this.ocpp.status.stations) {
				byId.set(station.id, {
					id: station.id,
					status: station.status,
					title: this.stationTitles[station.id],
				});
				published.add(station.id);
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
			// published stations first, then rule-only entries; each sorted by id
			const byIdAlpha = (a: StationEntry, b: StationEntry) => a.id.localeCompare(b.id);
			const all = [...byId.values()];
			return [
				...all.filter((e) => published.has(e.id)).sort(byIdAlpha),
				...all.filter((e) => !published.has(e.id)).sort(byIdAlpha),
			];
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

/* min-width:0 lets the identity actually truncate */
.station-identity {
	min-width: 0;
}
</style>
