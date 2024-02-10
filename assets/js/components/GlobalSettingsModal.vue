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
						<UserInterfaceSettings
							:sponsor="sponsor"
							:sponsorTokenExpires="sponsorTokenExpires"
						/>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import UserInterfaceSettings from "./Config/UserInterfaceSettings.vue";

export default {
	name: "GlobalSettingsModal",
	components: { UserInterfaceSettings },
	props: {
		sponsor: String,
		sponsorTokenExpires: Number,
	},
};
</script>
