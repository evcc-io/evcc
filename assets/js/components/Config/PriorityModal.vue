<template>
	<GenericModal
		id="priorityModal"
		ref="modal"
		:title="`${$t('config.priority.title')} 🧪`"
		config-modal-name="priority"
		data-testid="priority-modal"
		@open="open"
	>
		<p>{{ $t("config.priority.description") }}</p>
		<ErrorMessage :error="error" />
		<form ref="form" class="container mx-0 px-0" @submit.prevent="save">
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
import store from "@/store";
import api from "@/api";
import { PRIORITY_BASIS, PRIORITY_STRATEGY } from "@/types/evcc";
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
	components: { ErrorMessage, FormRow, GenericModal },
	emits: ["changed"],
	data() {
		return {
			saving: false,
			error: null as string | null,
			values: {} as Values,
			serverValues: {} as Values,
			strategies: Object.values(PRIORITY_STRATEGY),
			bases: Object.values(PRIORITY_BASIS),
		};
	},
	computed: {
		changed(): (keyof Values)[] {
			return (Object.keys(this.values) as (keyof Values)[]).filter(
				(key) => this.values[key] !== this.serverValues[key]
			);
		},
		nothingChanged(): boolean {
			return this.changed.length === 0;
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
		effectiveBasis(): PRIORITY_BASIS {
			const { priorityBasis, effectivePriorityBasis } = store?.state || {};
			// the site falls back to the percent basis when a loadpoint reports soc without a
			// known vehicle capacity, but that only describes the saved basis, not an edited one
			if (effectivePriorityBasis && this.values.priorityBasis === priorityBasis) {
				return effectivePriorityBasis;
			}
			return this.values.priorityBasis;
		},
		hysteresisUnit(): string {
			return this.effectiveBasis === PRIORITY_BASIS.ENERGY ? "kWh" : "%";
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
				this.$emit("changed");
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
