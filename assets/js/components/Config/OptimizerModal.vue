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
		<div v-if="enabled" class="my-3">
			<label class="form-label" for="optimizerAutomatic">
				{{ $t("config.optimizer.automatic") }}
			</label>
			<SelectGroup
				id="optimizerAutomatic"
				class="w-100"
				transparent
				equal-width
				:model-value="automatic"
				:options="levelOptions"
				@update:model-value="changeAutomatic"
			/>
			<div class="text-muted small">
				<p v-if="automatic === OPTIMIZER_AUTOMATIC.OFF" class="mt-2 mb-0">
					{{ $t("config.optimizer.automaticOff") }}
				</p>
				<template v-else>
					<p class="mt-2 mb-1">{{ $t("config.optimizer.automaticHint") }}</p>
					<ul class="mb-2 ps-3">
						<li>{{ $t("config.optimizer.automaticBattery") }}</li>
						<template v-if="automatic === OPTIMIZER_AUTOMATIC.FULL">
							<li>{{ $t("config.optimizer.automaticCharging") }}</li>
							<li>{{ $t("config.optimizer.automaticLimits") }}</li>
							<li>{{ $t("config.optimizer.automaticPlans") }}</li>
						</template>
					</ul>
					<p class="mb-1">{{ $t("config.optimizer.automaticNotControlled") }}</p>
					<ul class="mb-0 ps-3">
						<li v-if="automatic === OPTIMIZER_AUTOMATIC.BATTERY">
							{{ $t("config.optimizer.automaticNotControlledLoadpoints") }}
						</li>
						<template v-else>
							<li>{{ $t("config.optimizer.automaticNotControlledModes") }}</li>
							<li>{{ $t("config.optimizer.automaticNotControlledDevices") }}</li>
							<li>{{ $t("config.optimizer.automaticNotControlledVehicles") }}</li>
						</template>
					</ul>
				</template>
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
import SelectGroup from "../Helper/SelectGroup.vue";
import SponsorTokenRequired from "./DeviceModal/SponsorTokenRequired.vue";
import api from "@/api";
import store from "@/store";
import { docsPrefix } from "@/i18n";
import { OPTIMIZER_AUTOMATIC, type SelectOption } from "@/types/evcc";
import type { AxiosError } from "axios";

export default defineComponent({
	name: "OptimizerModal",
	components: { GenericModal, ErrorMessage, SelectGroup, SponsorTokenRequired },
	props: {
		isSponsor: Boolean,
	},
	data() {
		return {
			error: null as string | null,
			OPTIMIZER_AUTOMATIC,
		};
	},
	computed: {
		enabled(): boolean {
			return !!store.state?.optimizer;
		},
		automatic(): OPTIMIZER_AUTOMATIC {
			return store.state?.optimizerAutomatic || OPTIMIZER_AUTOMATIC.OFF;
		},
		levelOptions(): SelectOption<string>[] {
			return Object.values(OPTIMIZER_AUTOMATIC).map((value) => ({
				value,
				name: this.$t(`config.optimizer.automaticLevel.${value}`),
				disabled: !this.isSponsor,
			}));
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
		async changeAutomatic(level: string) {
			await this.post(`config/optimizerautomatic/${level}`);
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
