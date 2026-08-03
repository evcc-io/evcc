<template>
	<GenericModal
		id="priorityModal"
		ref="modal"
		:title="`${$t('config.priority.title')} 🧪`"
		config-modal-name="priority"
		data-testid="priority-modal"
		:autofocus="false"
		@open="open"
	>
		<p>{{ $t("config.priority.description") }}</p>
		<ErrorMessage :error="error" />
		<form ref="form" class="container mx-0 px-0" @submit.prevent="save">
			<FormRow
				v-if="loadpoints.length > 1"
				id="priorityOrder"
				:help="$t('config.priority.order.description')"
			>
				<PriorityOrder :key="tiersKey" v-model="tiers" :labels="loadpointLabels" />
			</FormRow>

			<hr v-if="loadpoints.length > 1" class="my-4" />
			<p>{{ $t("config.priority.descriptionStrategy") }}</p>

			<FormRow
				id="priorityStrategy"
				:label="$t('config.priority.labelStrategy')"
				:help="strategyDescription"
			>
				<select id="priorityStrategy" v-model="values.priorityStrategy" class="form-select">
					<option v-for="s in strategies" :key="s" :value="s">
						{{ $t(`config.priority.strategy.${s}`) }}
					</option>
				</select>
			</FormRow>

			<div v-show="strategyActive" class="ms-3">
				<FormRow
					id="priorityBasis"
					:label="$t('config.priority.labelBasis')"
					:help="basisDescription"
				>
					<select id="priorityBasis" v-model="values.priorityBasis" class="form-select">
						<option v-for="b in bases" :key="b" :value="b">
							{{ $t(`config.priority.basis.${b}`) }}
						</option>
					</select>
				</FormRow>

				<FormRow
					id="priorityHysteresis"
					:label="$t('config.priority.labelHysteresis')"
					:help="$t('config.priority.descriptionHysteresis')"
				>
					<div class="input-group input-width">
						<input
							id="priorityHysteresis"
							v-model="values.priorityHysteresis"
							type="number"
							step="1"
							min="0"
							max="99"
							required
							aria-describedby="priorityHysteresisUnit"
							class="form-control text-end"
						/>
						<span id="priorityHysteresisUnit" class="input-group-text">{{
							hysteresisUnit
						}}</span>
					</div>
				</FormRow>
			</div>

			<div class="mt-4 d-flex justify-content-between gap-2 flex-column flex-sm-row">
				<button
					type="button"
					class="btn btn-link text-muted btn-cancel"
					data-bs-dismiss="modal"
				>
					{{ $t("config.general.cancel") }}
				</button>

				<button
					type="submit"
					class="btn btn-primary order-1 order-sm-2 flex-grow-1 flex-sm-grow-0 px-4"
					:disabled="saving || nothingChanged"
				>
					<span
						v-if="saving"
						class="spinner-border spinner-border-sm"
						role="status"
						aria-hidden="true"
					></span>
					{{ $t("config.general.save") }}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import ErrorMessage from "../Helper/ErrorMessage.vue";
import FormRow from "./FormRow.vue";
import PriorityOrder from "./PriorityOrder.vue";
import store from "@/store";
import api from "@/api";
import { PRIORITY_BASIS, PRIORITY_STRATEGY, type Loadpoint } from "@/types/evcc";
import type { AxiosError } from "axios";

interface Values {
	priorityStrategy: PRIORITY_STRATEGY;
	priorityBasis: PRIORITY_BASIS;
	priorityHysteresis: number;
}

const ROUTES: Record<keyof Values, string> = {
	priorityStrategy: "prioritystrategy",
	priorityBasis: "prioritybasis",
	priorityHysteresis: "priorityhysteresis",
};

export default defineComponent({
	name: "PriorityModal",
	components: { ErrorMessage, FormRow, GenericModal, PriorityOrder },
	data() {
		return {
			saving: false,
			error: null as string | null,
			values: {} as Values,
			serverValues: {} as Values,
			tiers: {} as Record<string, number>,
			serverTiers: {} as Record<string, number>,
			tiersKey: 0,
			strategies: Object.values(PRIORITY_STRATEGY),
			bases: Object.values(PRIORITY_BASIS),
		};
	},
	computed: {
		loadpoints(): Loadpoint[] {
			return store.state?.loadpoints || [];
		},
		changed(): (keyof Values)[] {
			return (Object.keys(this.values) as (keyof Values)[]).filter(
				(key) => this.values[key] !== this.serverValues[key]
			);
		},
		changedTiers(): string[] {
			return Object.keys(this.tiers).filter(
				(name) => this.tiers[name] !== this.serverTiers[name]
			);
		},
		nothingChanged(): boolean {
			return this.changed.length === 0 && this.changedTiers.length === 0;
		},
		loadpointLabels(): Record<string, string> {
			return Object.fromEntries(
				this.loadpoints.map((lp, i) => [String(i), lp.title || `${i + 1}`])
			);
		},
		strategyDescription(): string {
			const strategy = this.values.priorityStrategy || PRIORITY_STRATEGY.NONE;
			return this.$t(`config.priority.strategyDescription.${strategy}`);
		},
		basisDescription(): string {
			const basis = this.values.priorityBasis || PRIORITY_BASIS.PERCENT;
			return this.$t(`config.priority.basisDescription.${basis}`);
		},
		strategyActive(): boolean {
			// basis and hysteresis only affect soc/deficit sub-ordering, not the none strategy
			return this.values.priorityStrategy !== PRIORITY_STRATEGY.NONE;
		},
		hysteresisUnit(): string {
			return this.values.priorityBasis === PRIORITY_BASIS.ENERGY ? "kWh" : "%";
		},
	},
	watch: {
		// loadpoints may arrive after the modal was deep-link opened
		"loadpoints.length"() {
			this.seedTiers();
		},
	},
	methods: {
		reset() {
			const { priorityStrategy, priorityBasis, priorityHysteresis } = store?.state || {};
			this.saving = false;
			this.error = null;
			this.values = {
				// fall back to the none/percent defaults
				priorityStrategy: priorityStrategy || PRIORITY_STRATEGY.NONE,
				priorityBasis: priorityBasis || PRIORITY_BASIS.PERCENT,
				priorityHysteresis: priorityHysteresis ?? 0,
			};
			this.serverValues = { ...this.values };
			this.seedTiers();
		},
		seedTiers() {
			this.tiers = Object.fromEntries(
				this.loadpoints.map((lp, i) => [String(i), lp.priority || 0])
			);
			this.serverTiers = { ...this.tiers };
			// remount the order component so it recalculates its visible lanes
			this.tiersKey++;
		},
		open() {
			this.reset();
		},
		async save() {
			this.saving = true;
			this.error = null;
			try {
				for (const key of this.changed) {
					await api.post(`${ROUTES[key]}/${encodeURIComponent(this.values[key])}`);
				}
				for (const name of this.changedTiers) {
					// runtime route uses the 1-based loadpoint position
					await api.post(`loadpoints/${Number(name) + 1}/priority/${this.tiers[name]}`);
				}
				(this.$refs["modal"] as any).close();
			} catch (err) {
				const axiosErr = err as AxiosError<{ error: string }>;
				this.error = axiosErr.response?.data?.error || axiosErr.message;
			}
			this.saving = false;
		},
	},
});
</script>
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
	padding-right: 0;
}
.input-width {
	width: 140px;
}
</style>
