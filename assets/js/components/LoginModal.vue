<template>
	<Teleport to="body">
		<div
			id="loginModal"
			class="modal fade text-dark"
			data-bs-backdrop="true"
			tabindex="-1"
			role="dialog"
			aria-hidden="false"
		>
			<div class="modal-dialog modal-dialog-centered open" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">{{ $t("loginModal.title") }}</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<FormRow id="loginPassword" :label="$t('loginModal.password')">
							<input id="loginPassword" v-model="password" class="form-control" />
						</FormRow>
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import FormRow from "./FormRow.vue";

export default {
	name: "LoginModal",
	components: { FormRow },
	data: () => {
		return {
			password: "",
		};
	},
};
</script>
