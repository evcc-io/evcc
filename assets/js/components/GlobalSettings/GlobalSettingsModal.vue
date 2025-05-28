<template>
	<Teleport to="body">
		<div
			id="globalSettingsModal"
			class="modal fade text-dark"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
		>
			<div class="modal-dialog modal-dialog-centered" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">{{ $t("settings.title") }}</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<UserInterfaceSettings :sponsor="sponsor" />
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import UserInterfaceSettings from "./UserInterfaceSettings.vue";
import type { Sponsor } from "@/types/evcc";

export default defineComponent({
	name: "GlobalSettingsModal",
	components: { UserInterfaceSettings },
	props: {
		sponsor: { type: Object as PropType<Sponsor>, default: () => ({}) },
	},
});
</script>
