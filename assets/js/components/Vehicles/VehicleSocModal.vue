<template>
	<GenericModal
		id="vehicleSocModal"
		ref="modal"
		:title="$t('main.vehicle.setSocTitle')"
		data-testid="vehicle-soc-modal"
		@open="prefill"
	>
		<div class="mb-3">
			<input
				v-model.number="socValue"
				type="number"
				min="0"
				max="100"
				step="1"
				class="form-control"
				:placeholder="$t('main.vehicle.setSoc')"
				@keyup.enter="confirm"
			/>
		</div>
		<p class="text-muted small mb-3">{{ $t("main.vehicle.setSocHelp") }}</p>
		<div class="d-flex justify-content-end">
			<button
				type="button"
				class="btn btn-primary"
				data-testid="vehicle-soc-confirm"
				@click="confirm"
			>
				{{ $t("main.vehicle.setSocConfirm") }}
			</button>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";

export default defineComponent({
	name: "VehicleSocModal",
	components: { GenericModal },
	props: {
		vehicleSoc: { type: Number, default: 0 },
		vehicleSocManual: Boolean,
	},
	emits: ["vehicle-soc-updated"],
	data() {
		return {
			socValue: null as number | null,
		};
	},
	methods: {
		open() {
			const modalRef = this.$refs["modal"] as InstanceType<typeof GenericModal> | undefined;
			modalRef?.open();
		},
		prefill() {
			this.socValue =
				this.vehicleSocManual && this.vehicleSoc ? Math.round(this.vehicleSoc) : null;
		},
		confirm() {
			const v = this.socValue;
			if (v == null || !Number.isFinite(v) || v < 0 || v > 100) return;
			this.$emit("vehicle-soc-updated", Math.round(v));
			const modalRef = this.$refs["modal"] as InstanceType<typeof GenericModal> | undefined;
			modalRef?.close();
		},
	},
});
</script>
