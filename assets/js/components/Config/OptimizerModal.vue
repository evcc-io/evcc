<template>
	<GenericModal
		id="optimizerModal"
		:title="`${$t('config.optimizer.title')} 🧪`"
		config-modal-name="optimizer"
		data-testid="optimizer-modal"
	>
		<p>
			{{ $t("config.optimizer.description") }}
			<a :href="docsLink" target="_blank">{{ $t("config.general.docsLink") }}</a>
		</p>
		<SponsorTokenRequired v-if="!isSponsor" feature class="mt-0" />
		<ErrorMessage :error="error" />
		<div class="form-check form-switch my-3">
			<input
				id="optimizerEnabled"
				:checked="enabled"
				class="form-check-input"
				type="checkbox"
				role="switch"
				:disabled="!isSponsor"
				@change="change"
			/>
			<div class="form-check-label">
				<label for="optimizerEnabled">
					{{ $t("config.optimizer.enable") }}
				</label>
			</div>
		</div>
		<div v-if="enabled" class="form-check form-switch my-3">
			<input
				id="optimizerAutomatic"
				:checked="automaticEnabled"
				class="form-check-input"
				type="checkbox"
				role="switch"
				:disabled="!isSponsor"
				@change="changeAutomatic"
			/>
			<div class="form-check-label">
				<label for="optimizerAutomatic">
					{{ $t("config.optimizer.automatic") }}
				</label>
				<div class="mt-3">
					<label class="form-label" for="optimizerControls">
						{{ $t("config.optimizer.controls") }}
					</label>
					<select
						id="optimizerControls"
						class="form-select"
						:value="level"
						:disabled="!automaticEnabled"
						@change="changeLevel"
					>
						<option v-for="value in levels" :key="value" :value="value">
							{{ $t(`config.optimizer.automaticLevel.${value}.label`) }}
						</option>
					</select>
					<div class="form-text evcc-gray">
						{{ $t(`config.optimizer.automaticLevel.${automatic}.description`) }}
					</div>
				</div>
			</div>
		</div>
		<p v-if="enabled && !hasEvopt" class="text-muted small mt-2">
			{{ $t("config.optimizer.info") }}
		</p>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import ErrorMessage from "../Helper/ErrorMessage.vue";
import SponsorTokenRequired from "./DeviceModal/SponsorTokenRequired.vue";
import api from "@/api";
import store from "@/store";
import { docsPrefix } from "@/i18n";
import { OPTIMIZER_AUTOMATIC } from "@/types/evcc";
import type { AxiosError } from "axios";

export default defineComponent({
	name: "OptimizerModal",
	components: { GenericModal, ErrorMessage, SponsorTokenRequired },
	props: {
		isSponsor: Boolean,
	},
	data() {
		return {
			error: null as string | null,
			levels: [OPTIMIZER_AUTOMATIC.BATTERY, OPTIMIZER_AUTOMATIC.FULL],
		};
	},
	computed: {
		enabled(): boolean {
			return !!store.state?.optimizer;
		},
		automatic(): OPTIMIZER_AUTOMATIC {
			return store.state?.optimizerAutomatic || OPTIMIZER_AUTOMATIC.OFF;
		},
		automaticEnabled(): boolean {
			return this.automatic !== OPTIMIZER_AUTOMATIC.OFF;
		},
		level(): OPTIMIZER_AUTOMATIC {
			return this.automaticEnabled ? this.automatic : OPTIMIZER_AUTOMATIC.BATTERY;
		},
		hasEvopt(): boolean {
			return !!store.state?.evopt;
		},
		docsLink(): string {
			return `${docsPrefix()}/features/optimizer`;
		},
	},
	methods: {
		async change(e: Event) {
			await this.post(`config/optimizer/${(e.target as HTMLInputElement).checked}`);
		},
		async changeAutomatic(e: Event) {
			const { checked } = e.target as HTMLInputElement;
			const level = checked ? OPTIMIZER_AUTOMATIC.BATTERY : OPTIMIZER_AUTOMATIC.OFF;
			await this.post(`config/optimizerautomatic/${level}`);
		},
		async changeLevel(e: Event) {
			await this.post(`config/optimizerautomatic/${(e.target as HTMLSelectElement).value}`);
		},
		async post(url: string) {
			try {
				this.error = null;
				await api.post(url);
			} catch (err) {
				const e = err as AxiosError<{ error: string }>;
				this.error = e.response?.data?.error || e.message;
			}
		},
	},
});
</script>
