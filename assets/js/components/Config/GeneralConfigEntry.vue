<template>
	<div
		class="d-flex justify-content-between flex-wrap align-items-center gap-2"
		:data-testid="testId"
	>
		<strong class="text-truncate d-flex align-items-center"
			>{{ label }}{{ experimental ? " 🧪" : "" }}</strong
		>
		<div class="d-flex align-items-center text-truncate">
			<div
				class="text-truncate align-items-center flex-grow-1 flex-shrink-1 text-end"
				:class="textClass"
			>
				<slot name="text-prefix"></slot>
				{{ text }}
			</div>
			<button
				class="btn btn-link flex-shrink-0"
				style="margin-right: -1rem; color: var(--evcc-text-default)"
				type="button"
				:title="$t('config.main.edit')"
				tabindex="0"
				@click.prevent="handleEditClick"
			>
				<EditIcon size="xs" />
			</button>
		</div>
	</div>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import EditIcon from "../MaterialIcon/Edit.vue";

export default {
	name: "GeneralConfigEntry",
	components: { EditIcon },
	props: {
		testId: { type: String, required: true },
		label: { type: String, required: true },
		text: { type: String, default: "---" },
		textClass: { type: String, default: "" },
		modalId: { type: String, required: true },
		experimental: { type: Boolean, default: false },
	},
	emits: ["edit-clicked"],
	methods: {
		handleEditClick() {
			this.$emit("edit-clicked");
			this.openModal();
		},
		openModal() {
			const $el = document.getElementById(this.modalId);
			if ($el) {
				Modal.getOrCreateInstance($el).show();
			} else {
				console.error(`modal ${this.modalId} not found`);
			}
		},
	},
};
</script>
