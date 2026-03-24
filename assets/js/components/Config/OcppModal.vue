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
			<div
				v-if="connectedStations.length === 0 && detectedStations.length === 0"
				class="text-muted"
			>
				{{ $t("config.ocpp.noChargers") }}
			</div>

			<!-- Connection status -->
			<div v-if="connectedStations.length > 0">
				<h6 class="mb-3">{{ $t("config.ocpp.connectionStatus") }}</h6>
				<p class="text-muted small mb-3">{{ $t("config.ocpp.connectionStatusHelp") }}</p>

				<ul class="list-group stations-list mb-4">
					<li
						v-for="station in connectedStations"
						:key="station.id"
						class="list-group-item d-flex justify-content-between align-items-center"
					>
						<code>{{ station.id }}</code>
						<span
							class="badge"
							:class="statusBadgeClass(station.status)"
							:title="$t(`config.ocpp.status.${station.status}`)"
							data-bs-toggle="tooltip"
						>
							{{ $t(`config.ocpp.status.${station.status}`) }}
						</span>
					</li>
				</ul>
			</div>

			<!-- Detected chargers -->
			<div v-if="detectedStations.length > 0">
				<h6 class="mb-3">{{ $t("config.ocpp.detectedChargers") }}</h6>
				<p class="text-muted small mb-3">{{ $t("config.ocpp.detectedHelp") }}</p>

				<ul class="list-group stations-list">
					<li
						v-for="station in detectedStations"
						:key="station.id"
						class="list-group-item d-flex justify-content-between align-items-center"
					>
						<code>{{ station.id }}</code>
						<span class="badge bg-secondary">
							{{ $t(`config.ocpp.status.${station.status}`) }}
						</span>
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
import type { Ocpp, OcppStationStatus } from "@/types/evcc";
import { getOcppUrl, getOcppUrlWithStationId } from "@/utils/ocpp";

export default defineComponent({
	name: "OcppModal",
	components: {
		GenericModal,
		FormRow,
		Markdown,
	},
	props: {
		ocpp: {
			type: Object as PropType<Ocpp>,
			default: () => ({ config: { port: 0 }, status: { stations: [] } }),
		},
	},
	computed: {
		status() {
			return this.ocpp.status;
		},
		stations() {
			return this.status.stations;
		},
		ocppUrl(): string {
			return getOcppUrl(this.ocpp);
		},
		ocppUrlWithStationId(): string {
			return getOcppUrlWithStationId(this.ocpp);
		},
		connectedStations(): OcppStationStatus[] {
			return this.stations.filter(
				(s) => s.status === "connected" || s.status === "configured"
			);
		},
		detectedStations(): OcppStationStatus[] {
			return this.stations.filter((s) => s.status === "unknown");
		},
	},
	methods: {
		statusBadgeClass(status: string): string {
			switch (status) {
				case "connected":
					return "bg-success";
				case "configured":
					return "bg-warning";
				case "unknown":
					return "bg-secondary";
				default:
					return "bg-secondary";
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

.list-group-item code {
	font-size: 0.9rem;
}
</style>
