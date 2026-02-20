<template>
	<Teleport to="body">
		<div
			:id="id"
			ref="modal"
			:class="['modal', 'fade', 'text-dark', fadeClass]"
			tabindex="-1"
			role="dialog"
			:aria-hidden="isModalVisible ? 'false' : 'true'"
			:data-bs-backdrop="uncloseable ? 'static' : 'true'"
			:data-bs-keyboard="uncloseable ? 'false' : 'true'"
			:data-testid="dataTestid"
		>
			<div class="modal-dialog modal-dialog-centered" :class="sizeClass" role="document">
				<div class="modal-content">
					<div class="modal-header d-flex justify-content-between align-items-center">
						<h5 class="modal-title">
							{{ title }}
						</h5>
						<div class="d-flex align-items-center gap-1">
							<slot name="header-actions"></slot>
							<button
								v-if="!uncloseable"
								type="button"
								class="btn-close"
								data-bs-dismiss="modal"
								aria-label="Close"
							></button>
						</div>
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
import { registerModal, unregisterModal, onModalHidden, getModalFade } from "@/configModal";

export default defineComponent({
	name: "GenericModal",
	props: {
		id: String,
		title: String,
		dataTestid: String,
		uncloseable: Boolean,
		size: String,
		autofocus: { type: Boolean, default: true },
		configModalName: String,
	},
	emits: ["open", "opened", "close", "closed", "visibilitychange"],
	data() {
		return {
			isModalVisible: false,
		};
	},
	computed: {
		sizeClass() {
			return this.size ? `modal-${this.size}` : "";
		},
		fadeClass(): string {
			const fade = this.configModalName && getModalFade(this.configModalName);
			return fade ? `fade-${fade}` : "";
		},
	},
	mounted() {
		this.$refs["modal"]?.addEventListener("show.bs.modal", this.handleShow);
		this.$refs["modal"]?.addEventListener("shown.bs.modal", this.handleShown);
		this.$refs["modal"]?.addEventListener("hide.bs.modal", this.handleHide);
		this.$refs["modal"]?.addEventListener("hidden.bs.modal", this.handleHidden);
		document.addEventListener("visibilitychange", this.handleVisibilityChange);
		if (this.configModalName) {
			registerModal(this.configModalName, this.$refs["modal"] as HTMLElement);
		}
	},
	unmounted() {
		this.$refs["modal"]?.removeEventListener("show.bs.modal", this.handleShow);
		this.$refs["modal"]?.removeEventListener("shown.bs.modal", this.handleShown);
		this.$refs["modal"]?.removeEventListener("hide.bs.modal", this.handleHide);
		this.$refs["modal"]?.removeEventListener("hidden.bs.modal", this.handleHidden);
		document.removeEventListener("visibilitychange", this.handleVisibilityChange);
		if (this.configModalName) {
			unregisterModal(this.configModalName);
		}
	},
	methods: {
		handleShow() {
			this.$emit("open");
		},
		handleShown() {
			this.$emit("opened");
			if (this.autofocus) {
				this.$nextTick(() => {
					const firstInput =
						this.$refs["modalBody"]?.querySelector("input, select, button");
					if (firstInput instanceof HTMLElement) {
						firstInput.focus();
					}
				});
			}
			this.isModalVisible = true;
		},
		handleHide() {
			this.$emit("close");
		},
		handleHidden() {
			this.$emit("closed");
			this.isModalVisible = false;
			if (this.configModalName) {
				onModalHidden(this.configModalName);
			}
		},
		open() {
			const modal = this.$refs["modal"] as HTMLElement;
			Modal.getOrCreateInstance(modal).show();
		},
		close() {
			const modal = this.$refs["modal"] as HTMLElement;
			Modal.getOrCreateInstance(modal).hide();
		},
		handleVisibilityChange() {
			if (document.visibilityState === "visible" && this.isModalVisible) {
				this.$emit("visibilitychange");
			}
		},
	},
});
</script>
