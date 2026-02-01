<template>
	<button
		ref="tooltip"
		type="button"
		class="btn btn-sm btn-outline-secondary position-relative border-0 p-2 edit-button"
		:class="{ 'opacity-25': !editable, invisible: noEditButton }"
		data-bs-toggle="tooltip"
		data-bs-html="true"
		:title="tooltipTitle"
		:aria-label="editable ? $t('config.main.edit') : null"
		:disabled="!editable || noEditButton"
		@click="edit"
	>
		<shopicon-regular-adjust size="s"></shopicon-regular-adjust>
	</button>
</template>

<script>
import "@h2d2/shopicons/es/regular/adjust";
import Tooltip from "bootstrap/js/dist/tooltip";

export default {
	name: "DeviceCardEditIcon",
	props: {
		name: String,
		editable: Boolean,
		noEditButton: Boolean,
	},
	emits: ["edit"],
	data() {
		return {
			tooltip: null,
		};
	},
	computed: {
		tooltipTitle() {
			if (!this.name) {
				return "";
			}
			let title = `${this.$t("config.main.name")}: <span class='font-monospace'>${this.name}</span>`;
			if (!this.editable) {
				title += `<div class="mt-1">${this.$t("config.main.yaml")}</div>`;
			}
			return `<div class="text-start">${title}</div>`;
		},
	},
	watch: {
		tooltipTitle() {
			this.initTooltip();
		},
	},
	mounted() {
		this.initTooltip();
	},
	methods: {
		edit() {
			if (this.editable) {
				this.tooltip?.hide();
				this.$emit("edit");
			}
		},
		initTooltip() {
			this.$nextTick(() => {
				this.tooltip?.dispose();
				if (this.$refs.tooltip) {
					this.tooltip = new Tooltip(this.$refs.tooltip);
				}
			});
		},
	},
};
</script>

<style scoped>
.edit-button {
	/* transparent button, right align icon */
	margin-right: -0.5rem;
}
button:disabled {
	pointer-events: auto;
}
</style>
