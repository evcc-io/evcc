<template>
	<GenericModal
		id="logLevelsModal"
		ref="modal"
		data-testid="loglevels-modal"
		:title="$t('config.logLevels.title')"
		:prevent-dismiss="changed"
		@open="reset"
	>
		<p>{{ $t("config.logLevels.description") }}</p>
		<p v-if="error" class="text-danger">{{ error }}</p>
		<form class="container mx-0 px-0" @submit.prevent="save">
			<FormRow id="logLevelsDefault" :label="$t('config.logLevels.defaultLabel')">
				<select id="logLevelsDefault" v-model="defaultLevel" class="form-select level">
					<option v-for="level in LOG_LEVELS" :key="level" :value="level.toUpperCase()">
						{{ level.toUpperCase() }}
					</option>
				</select>
			</FormRow>

			<label class="form-label">{{ $t("config.logLevels.overridesLabel") }}</label>
			<div v-for="(entry, index) in entries" :key="entry.id" class="d-flex gap-2 mb-3">
				<select
					v-model="entry.area"
					class="form-select"
					:class="{ 'border-warning': unavailable(entry.area) }"
					:aria-label="$t('config.logLevels.areaLabel')"
				>
					<option value="" disabled>{{ $t("config.logLevels.areaPlaceholder") }}</option>
					<option v-for="area in areaOptions(entry)" :key="area" :value="area">
						{{ areaText(area) }}
					</option>
				</select>
				<select
					v-model="entry.level"
					class="form-select level flex-shrink-0"
					:aria-label="$t('config.logLevels.levelLabel')"
				>
					<option value="" disabled>{{ $t("config.logLevels.levelPlaceholder") }}</option>
					<option v-for="level in LOG_LEVELS" :key="level" :value="level.toUpperCase()">
						{{ level.toUpperCase() }}
					</option>
				</select>
				<button
					type="button"
					class="btn btn-sm btn-outline-secondary border-0 evcc-gray flex-shrink-0"
					:aria-label="$t('config.general.remove')"
					tabindex="0"
					@click="entries.splice(index, 1)"
				>
					<shopicon-regular-trash size="s" class="icon"></shopicon-regular-trash>
				</button>
			</div>

			<button
				type="button"
				class="d-flex btn btn-sm btn-outline-secondary border-0 evcc-gray align-items-center gap-2 mb-3"
				tabindex="0"
				@click="addEntry"
			>
				<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
				{{ $t("config.logLevels.add") }}
			</button>

			<div class="mt-4 d-flex justify-content-between gap-2 flex-column flex-sm-row">
				<div
					class="d-flex justify-content-between order-2 order-sm-1 gap-2 flex-grow-1 flex-sm-grow-0"
				>
					<button type="button" class="btn btn-link text-muted" data-bs-dismiss="modal">
						{{ $t("config.general.cancel") }}
					</button>
					<button
						type="button"
						class="btn btn-link text-danger"
						:disabled="busy"
						@click="resetToDefaults"
					>
						{{ $t("config.logLevels.reset") }}
					</button>
				</div>
				<button
					type="submit"
					class="btn btn-primary order-1 order-sm-2 flex-grow-1 flex-sm-grow-0 px-4"
					:disabled="busy || !changed || !complete"
				>
					{{ $t("config.general.save") }}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/plus";
import "@h2d2/shopicons/es/regular/trash";
import { defineComponent, type PropType } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import FormRow from "./FormRow.vue";
import api from "@/api";
import { LOG_LEVELS } from "@/utils/log";
import type { LogLevels } from "@/types/evcc";

interface LogLevelEntry {
	id: number;
	area: string;
	level: string;
}

let lastId = 0;

const sortAreas = (areas: string[]) =>
	[...areas].sort((a, b) => a.localeCompare(b, undefined, { numeric: true }));

export default defineComponent({
	name: "LogLevelsModal",
	components: { GenericModal, FormRow },
	props: {
		logLevels: { type: Object as PropType<LogLevels>, default: () => ({}) },
	},
	data() {
		return {
			LOG_LEVELS,
			availableAreas: [] as string[],
			defaultLevel: "INFO",
			entries: [] as LogLevelEntry[],
			busy: false,
			error: "",
		};
	},
	computed: {
		payload(): LogLevels {
			const levels: Record<string, string> = {};
			for (const { area, level } of this.entries) {
				if (area) levels[area] = level;
			}
			return { default: this.defaultLevel, levels };
		},
		complete(): boolean {
			return this.entries.every(({ area, level }) => area && level);
		},
		changed(): boolean {
			const current = this.logLevels;
			const levels = current.levels || {};
			const payload = this.payload;
			return (
				payload.default !== (current.default || "INFO") ||
				JSON.stringify(sortedEntries(payload.levels)) !==
					JSON.stringify(sortedEntries(levels))
			);
		},
	},
	methods: {
		open() {
			(this.$refs["modal"] as unknown as InstanceType<typeof GenericModal>).open();
		},
		async reset() {
			this.error = "";
			this.defaultLevel = this.logLevels.default || "INFO";
			this.entries = sortAreas(Object.keys(this.logLevels.levels || {})).map((area) => ({
				id: ++lastId,
				area,
				level: this.logLevels.levels[area] ?? this.defaultLevel,
			}));
			try {
				const response = await api.get("/system/log/areas");
				this.availableAreas = response.data || [];
			} catch (e) {
				console.error(e);
			}
		},
		unavailable(area: string) {
			return !!area && this.availableAreas.length > 0 && !this.availableAreas.includes(area);
		},
		areaText(area: string) {
			return this.unavailable(area)
				? this.$t("config.logLevels.areaUnavailable", { area })
				: area;
		},
		areaOptions(entry: LogLevelEntry) {
			const used = new Set(this.entries.map((e) => e.area));
			return sortAreas(
				[...new Set([...this.availableAreas, entry.area])].filter(
					(area) => area && (area === entry.area || !used.has(area))
				)
			);
		},
		addEntry() {
			this.entries.push({ id: ++lastId, area: "", level: "" });
		},
		async submit(request: () => Promise<{ status: number; data: { error?: string } }>) {
			this.busy = true;
			this.error = "";
			try {
				const res = await request();
				if (res.status === 200) {
					(this.$refs["modal"] as unknown as InstanceType<typeof GenericModal>).close();
				} else {
					this.error = res.data.error || "";
				}
			} catch (e) {
				console.error(e);
			}
			this.busy = false;
		},
		save() {
			return this.submit(() =>
				api.post("/config/logLevels", this.payload, {
					validateStatus: (s) => [200, 400].includes(s),
				})
			);
		},
		resetToDefaults() {
			return this.submit(() =>
				api.delete("/config/logLevels", { validateStatus: (s) => [200, 400].includes(s) })
			);
		},
	},
});

function sortedEntries(levels: Record<string, string>) {
	return Object.entries(levels).sort(([a], [b]) => a.localeCompare(b));
}
</script>

<style scoped>
.level {
	width: 8rem;
}
</style>
