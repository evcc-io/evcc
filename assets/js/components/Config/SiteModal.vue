<template>
	<GenericModal
		id="siteModal"
		ref="modal"
		:title="$t('config.site.title')"
		data-testid="site-modal"
		config-modal-name="site"
		@open="open"
	>
		<SiteForm ref="form" @saved="saved">
			<template #default="{ saving, changed }">
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
						:disabled="saving || !changed"
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
			</template>
		</SiteForm>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../Helper/GenericModal.vue";
import SiteForm from "./SiteForm.vue";

export default defineComponent({
	name: "SiteModal",
	components: { GenericModal, SiteForm },
	emits: ["changed"],
	methods: {
		open() {
			(this.$refs["form"] as unknown as InstanceType<typeof SiteForm>).load();
		},
		saved() {
			this.$emit("changed");
			(this.$refs["modal"] as unknown as InstanceType<typeof GenericModal>).close();
		},
	},
});
</script>
