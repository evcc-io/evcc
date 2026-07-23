<template>
	<GenericModal
		id="batteryModal"
		:title="`${$t('config.battery.title')} 🧪`"
		config-modal-name="battery"
		data-testid="battery-modal"
	>
		<p>{{ $t("config.battery.description") }}</p>
		<ErrorMessage :error="error" />
		<div class="form-check form-switch my-3">
			<input
				id="batteryGridDischarge"
				:checked="gridDischarge"
				class="form-check-input"
				type="checkbox"
				role="switch"
				@change="change"
			/>
			<div class="form-check-label">
				<label for="batteryGridDischarge">
					{{ $t("config.battery.gridDischarge") }}
				</label>
			</div>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import ErrorMessage from "../Helper/ErrorMessage.vue";
import api from "@/api";
import type { AxiosError } from "axios";

export default defineComponent({
	name: "BatteryModal",
	components: { GenericModal, ErrorMessage },
	props: {
		gridDischarge: Boolean,
	},
	data() {
		return {
			error: null as string | null,
		};
	},
	methods: {
		async change(e: Event) {
			const target = e.target as HTMLInputElement;
			try {
				this.error = null;
				await api.post(`batterygriddischarge/${target.checked}`);
			} catch (err) {
				target.checked = this.gridDischarge; // revert on failure to stay in sync with state
				const axiosErr = err as AxiosError<{ error: string }>;
				this.error = axiosErr.response?.data?.error || axiosErr.message;
			}
		},
	},
});
</script>
