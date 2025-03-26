<template>
	<Teleport to="body">
		<div
			:id="id"
			ref="modal"
			:class="classes"
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
					<div ref="modalBody" class="modal-body">
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
	emits: ["open", "opened", "close", "closed"],
	data() {
		return {
			isModalVisible: false,
		};
	},
	computed: {
		classes() {
			return [
				"modal",
				"fade",
				"text-dark",
				{ show: this.isModalVisible },
				this.sizeClass,
				this.fadeClass,
			];
		},
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
	mounted() {
		this.$refs.modal.addEventListener("show.bs.modal", this.handleShow);
		this.$refs.modal.addEventListener("shown.bs.modal", this.handleShown);
		this.$refs.modal.addEventListener("hide.bs.modal", this.handleHide);
		this.$refs.modal.addEventListener("hidden.bs.modal", this.handleHidden);
	},
	unmounted() {
		this.$refs.modal?.removeEventListener("show.bs.modal", this.handleShow);
		this.$refs.modal?.removeEventListener("shown.bs.modal", this.handleShown);
		this.$refs.modal?.removeEventListener("hide.bs.modal", this.handleHide);
		this.$refs.modal?.removeEventListener("hidden.bs.modal", this.handleHidden);
	},
	methods: {
		handleShow() {
			this.$emit("open");
		},
		handleShown() {
			this.$emit("opened");
			// focus first input or select
			this.$nextTick(() => {
				const firstInput = this.$refs.modalBody.querySelector("input, select, button");
				if (firstInput) {
					firstInput.focus();
				}
			});
			this.isModalVisible = true;
		},
		handleHide() {
			console.log("GenericModal: hide >", this.id);
			this.$emit("close");
			console.log("GenericModal: hide <", this.id);
		},
		handleHidden() {
			console.log("GenericModal: hidden >", this.id);
			this.$emit("closed");
			this.isModalVisible = false;
			console.log("GenericModal: hidden <", this.id);
		},
		open() {
			console.log("GenericModal: open >", this.id);
			this.$nextTick(() => {
				Modal.getOrCreateInstance(this.$refs.modal).show();
				console.log("GenericModal: open <", this.id);
			});
		},
		close() {
			console.log("GenericModal: close >", this.id);
			this.$nextTick(() => {
				Modal.getOrCreateInstance(this.$refs.modal).hide();
				console.log("GenericModal: close <", this.id);
			});
		},
	},
};
</script>
