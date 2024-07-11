<template>
	<Teleport to="body">
		<div
			:id="id"
			ref="modal"
			class="modal fade text-dark"
			:class="[sizeClass, fadeClass]"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
			:data-bs-backdrop="uncloseable ? 'static' : 'true'"
			:data-bs-keyboard="uncloseable ? 'false' : 'true'"
			:data-testid="dataTestid"
		>
			<div class="modal-dialog modal-dialog-centered" role="document">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							{{ title }}
						</h5>
						<button
							v-if="!uncloseable"
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<slot />
					</div>
				</div>
			</div>
		</div>
	</Teleport>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";

export default {
	name: "GenericModal",
	props: {
		id: String,
		title: String,
		dataTestid: String,
		uncloseable: Boolean,
		fade: String,
		size: String,
	},
	emits: ["open", "closed"],
	mounted() {
		this.$refs.modal.addEventListener("show.bs.modal", this.handleShow);
		this.$refs.modal.addEventListener("hidden.bs.modal", this.handleHidden);
	},
	unmounted() {
		this.$refs.modal?.removeEventListener("show.bs.modal", this.handleShow);
		this.$refs.modal?.removeEventListener("hidden.bs.modal", this.handleHidden);
	},
	computed: {
		sizeClass() {
			return this.size ? `modal-${this.size}` : "";
		},
		fadeClass() {
			if (this.fade) {
				return `fade-${this.fade}`;
			}
			return "";
		},
	},
	methods: {
		handleShow() {
			this.$emit("open");
		},
		handleHidden() {
			this.$emit("closed");
		},
		open() {
			Modal.getOrCreateInstance(this.$refs.modal).show();
		},
		close() {
			Modal.getOrCreateInstance(this.$refs.modal).hide();
		},
	},
};
</script>
