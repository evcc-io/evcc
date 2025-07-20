<template>
	<div class="root round-box" :class="{ 'round-box--error': error }">
		<div class="d-flex align-items-center mb-2">
			<div class="icon me-2">
				<slot name="icon" />
			</div>
			<strong
				class="flex-grow-1 text-nowrap text-truncate"
				data-bs-toggle="tooltip"
				:title="name"
				>{{ title }}</strong
			>
			<button
				ref="tooltip"
				type="button"
				class="btn btn-sm btn-outline-secondary position-relative border-0 p-2"
				:class="{ 'opacity-25': !editable }"
				data-bs-toggle="tooltip"
				data-bs-html="true"
				:title="tooltipTitle"
				:aria-label="editable ? $t('config.main.edit') : null"
				:disabled="!editable"
				@click="edit"
			>
				<shopicon-regular-adjust size="s"></shopicon-regular-adjust>
			</button>
		</div>
		<hr class="my-3 divide" />
		<slot name="tags" />
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/adjust";
import "@h2d2/shopicons/es/regular/invoice";
import Tooltip from "bootstrap/js/dist/tooltip";

export default {
	name: "DeviceCard",
	props: {
		name: String,
		title: String,
		editable: Boolean,
		error: Boolean,
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
.root {
	display: block;
	list-style-type: none;
	border-radius: 1rem;
	padding: 1rem 1.5rem;
	min-height: 8rem;
}
.icon:empty {
	display: none;
}
.divide {
	margin-left: -1.5rem;
	margin-right: -1.5rem;
}
button:disabled {
	pointer-events: auto;
}
</style>
