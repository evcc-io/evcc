<template>
	<div class="mb-5" :data-testid="`${id}-additional-item`">
		<div class="d-flex justify-content-between align-items-center mb-2">
			<div class="d-flex align-items-baseline gap-2">
				<strong>{{ title }}</strong>
				<button
					type="button"
					class="btn btn-link btn-sm p-0 text-muted content"
					:class="{ 'content--dimmed': !included }"
					@click="openModal"
				>
					{{ $t("issue.additional.showDetails") }}
				</button>
			</div>
			<div class="form-check form-switch mb-0">
				<input
					:id="id"
					v-model="isIncluded"
					class="form-check-input"
					:class="switchClass"
					type="checkbox"
					role="switch"
				/>
				<label class="form-check-label" :for="id">
					{{ $t("issue.additional.include") }}
				</label>
			</div>
		</div>

		<div class="content" :class="{ 'content--dimmed': !included }">
			<slot name="description" :openModal="openModal"></slot>
		</div>

		<!-- Modal using GenericModal -->
		<GenericModal
			:id="modalId"
			:title="title"
			size="lg"
			:autofocus="false"
			:data-testid="`${id}-modal`"
			@closed="modalClosed"
		>
			<!-- Custom controls slot inside modal -->
			<slot name="controls"></slot>

			<textarea
				ref="textarea"
				:value="localContent"
				class="form-control font-monospace textarea--tiny"
				:rows="textareaRows"
				style="white-space: pre; overflow-wrap: normal; resize: vertical"
				@input="handleLocalInput"
			></textarea>

			<div class="d-flex justify-content-end mt-3">
				<button
					v-if="hasChanges"
					type="button"
					class="btn btn-primary"
					@click="applyAndClose"
				>
					{{ $t("config.general.applyAndClose") }}
				</button>
				<button v-else type="button" class="btn btn-secondary" @click="closeModal">
					{{ $t("config.general.close") }}
				</button>
			</div>
		</GenericModal>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Modal from "bootstrap/js/dist/modal";
import GenericModal from "../Helper/GenericModal.vue";
import type { HelpType } from "./types";

export default defineComponent({
	name: "IssueAdditionalItem",
	components: {
		GenericModal,
	},
	props: {
		title: { type: String, required: true },
		id: { type: String, required: true },
		included: { type: Boolean, required: true },
		content: { type: String, default: "" },
		description: { type: String, default: "" },
		helpType: { type: String as PropType<HelpType> },
	},
	emits: ["update:included", "update:content"],
	data() {
		return {
			localContent: "",
		};
	},
	computed: {
		isIncluded: {
			get(): boolean {
				return this.included;
			},
			set(value: boolean) {
				this.$emit("update:included", value);
			},
		},
		modalId(): string {
			return `${this.id}Modal`;
		},
		textareaRows(): number {
			if (!this.localContent) return 26;
			const lines = this.localContent.split("\n").length;
			return Math.max(26, lines + 1);
		},
		hasChanges(): boolean {
			return this.localContent !== this.content;
		},
		switchClass(): string {
			return this.included && this.helpType === "issue" ? "bg-danger border-danger" : "";
		},
	},
	watch: {
		content(newContent: string) {
			this.localContent = newContent;
		},
	},
	methods: {
		openModal() {
			this.localContent = this.content;
			const modalElement = document.getElementById(this.modalId) as HTMLElement;
			if (modalElement) {
				Modal.getOrCreateInstance(modalElement).show();
			}
		},
		modalClosed() {
			// Reset local content if there are unsaved changes
			this.localContent = this.content;
		},
		handleLocalInput(event: Event) {
			const target = event.target as HTMLTextAreaElement;
			this.localContent = target.value;
		},
		applyAndClose() {
			this.$emit("update:content", this.localContent);
			this.closeModal();
		},
		closeModal() {
			const modalElement = document.getElementById(this.modalId) as HTMLElement;
			if (modalElement) {
				Modal.getOrCreateInstance(modalElement).hide();
			}
		},
	},
});
</script>

<style scoped>
.content {
	opacity: 1;
	transition: opacity 0.3s ease;
}

.content--dimmed {
	opacity: 0.5;
}
</style>
