<template>
	<GenericModal
		:id="id"
		ref="modal"
		:title="title"
		:data-testid="dataTestid"
		:autofocus="false"
		@closed="resolve(false)"
	>
		<p>{{ description }}</p>
		<slot />
		<div class="mt-4 d-flex justify-content-between gap-2 flex-column flex-sm-row">
			<button type="button" class="btn btn-link text-muted" data-bs-dismiss="modal">
				{{ $t("general.cancel") }}
			</button>
			<button
				type="button"
				class="btn order-1 order-sm-2 flex-grow-1 flex-sm-grow-0 px-4"
				:class="danger ? 'btn-danger' : 'btn-primary'"
				@click="resolve(true)"
			>
				{{ confirmLabel }}
			</button>
		</div>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "./GenericModal.vue";

export default defineComponent({
	name: "ConfirmModal",
	components: { GenericModal },
	props: {
		id: { type: String, required: true },
		title: { type: String, required: true },
		description: { type: String, required: true },
		confirmLabel: { type: String, required: true },
		danger: Boolean,
		dataTestid: String,
	},
	data() {
		return {
			resolver: null as ((confirmed: boolean) => void) | null,
		};
	},
	methods: {
		confirm(): Promise<boolean> {
			const modal = this.$refs["modal"] as InstanceType<typeof GenericModal> | undefined;
			if (!modal) {
				return Promise.resolve(false);
			}
			return new Promise((resolve) => {
				this.resolver = resolve;
				modal.open();
			});
		},
		resolve(confirmed: boolean) {
			this.resolver?.(confirmed);
			this.resolver = null;
			if (confirmed) {
				this.close();
			}
		},
		close() {
			const modal = this.$refs["modal"] as InstanceType<typeof GenericModal> | undefined;
			modal?.close();
		},
	},
});
</script>
