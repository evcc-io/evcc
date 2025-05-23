<template>
	<Teleport to="body">
		<div
			:id="id"
			ref="modal"
			:class="classes"
			tabindex="-1"
			role="dialog"
			:aria-hidden="isModalVisible ? 'false' : 'true'"
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

<script lang="ts">
import Modal from "bootstrap/js/dist/modal";
import { defineComponent } from "vue";

export default defineComponent({
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
		this.$refs["modal"]?.addEventListener("show.bs.modal", this.handleShow);
		this.$refs["modal"]?.addEventListener("shown.bs.modal", this.handleShown);
		this.$refs["modal"]?.addEventListener("hide.bs.modal", this.handleHide);
		this.$refs["modal"]?.addEventListener("hidden.bs.modal", this.handleHidden);
	},
	unmounted() {
		this.$refs["modal"]?.removeEventListener("show.bs.modal", this.handleShow);
		this.$refs["modal"]?.removeEventListener("shown.bs.modal", this.handleShown);
		this.$refs["modal"]?.removeEventListener("hide.bs.modal", this.handleHide);
		this.$refs["modal"]?.removeEventListener("hidden.bs.modal", this.handleHidden);
	},
	methods: {
		handleShow() {
			console.log(this.dataTestid, "> show");
			this.$emit("open");
		},
		handleShown() {
			console.log(this.dataTestid, "> shown");
			this.$emit("opened");
			// focus first input or select
			this.$nextTick(() => {
				const firstInput = this.$refs["modalBody"]?.querySelector("input, select, button");
				if (firstInput instanceof HTMLElement) {
					firstInput.focus();
				}
			});
			this.isModalVisible = true;
		},
		handleHide() {
			console.log(this.dataTestid, "> hide");
			this.$emit("close");
		},
		handleHidden() {
			console.log(this.dataTestid, "> hidden");
			this.$emit("closed");
			this.isModalVisible = false;
		},
		open() {
			const modal = this.$refs["modal"] as HTMLElement;
			// @ts-expect-error bs internal
			console.log(this.dataTestid, "> open", modal._isShown);
			Modal.getOrCreateInstance(modal).show();
		},
		close() {
			const modal = this.$refs["modal"] as HTMLElement;
			// @ts-expect-error bs internal
			console.log(this.dataTestid, "> close", modal._isShown);
			Modal.getOrCreateInstance(modal).hide();
		},
	},
});
</script>
