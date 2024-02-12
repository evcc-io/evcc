<template>
	<li class="root py-2 px-4">
		<div class="d-flex align-items-center mb-2">
			<div class="icon me-2">
				<slot name="icon" />
			</div>
			<strong class="flex-grow-1 text-nowrap text-truncate">{{ name }}</strong>
			<button
				v-if="unconfigured"
				type="button"
				class="btn btn-sm btn-outline-secondary position-relative border-0 p-2"
				:title="$t('config.main.new')"
				@click="$emit('configure')"
			>
				<shopicon-regular-adjust size="s"></shopicon-regular-adjust>
			</button>
			<button
				v-else-if="editable"
				type="button"
				class="btn btn-sm btn-outline-secondary position-relative border-0 p-2"
				:title="$t('config.main.edit')"
				@click="$emit('edit')"
			>
				<shopicon-regular-adjust size="s"></shopicon-regular-adjust>
			</button>
			<button
				v-else
				ref="tooltip"
				type="button"
				class="btn btn-sm btn-outline-secondary position-relative border-0 p-2 opacity-25"
				data-bs-toggle="tooltip"
				:title="$t('config.main.yaml')"
			>
				<shopicon-regular-adjust size="s"></shopicon-regular-adjust>
			</button>
		</div>
		<div v-if="unconfigured" class="text-center py-3 evcc-gray">
			{{ $t("config.main.unconfigured") }}
		</div>
		<slot v-else name="tags" />
	</li>
</template>

<script>
import "@h2d2/shopicons/es/regular/adjust";
import "@h2d2/shopicons/es/regular/invoice";
import "@h2d2/shopicons/es/regular/edit";
import Tooltip from "bootstrap/js/dist/tooltip";

export default {
	name: "DeviceCard",
	props: {
		name: String,
		editable: Boolean,
		unconfigured: Boolean,
	},
	data() {
		return {
			tooltip: null,
		};
	},
	emits: ["edit", "configure"],
	mounted() {
		this.initTooltip();
	},
	watch: {
		editable() {
			this.initTooltip();
		},
		unconfigured() {
			this.initTooltip();
		},
	},
	methods: {
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
	min-height: 9rem;
	color: var(--evcc-default-text);
	border-radius: 1rem;
	border: 1px solid var(--evcc-gray-50);
	padding: 1rem 1rem 0.5rem;
	transition: border-color var(--evcc-transition-fast) linear;
	background: var(--evcc-box);
}
.root:hover {
	border-color: var(--evcc-gray);
}
.icon:empty {
	display: none;
}
</style>
