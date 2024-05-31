<template>
	<li class="root round-box" :class="{ 'round-box--error': error }">
		<div class="d-flex align-items-center mb-2">
			<div class="icon me-2">
				<slot name="icon" />
			</div>
			<strong class="flex-grow-1 text-nowrap text-truncate">{{ name }}</strong>
			<button
				v-if="editable"
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
		<hr class="my-3 divide" />
		<slot name="tags" />
	</li>
</template>

<script>
import "@h2d2/shopicons/es/regular/adjust";
import "@h2d2/shopicons/es/regular/invoice";
import Tooltip from "bootstrap/js/dist/tooltip";

export default {
	name: "DeviceCard",
	props: {
		name: String,
		editable: Boolean,
		error: Boolean,
	},
	data() {
		return {
			tooltip: null,
		};
	},
	emits: ["edit"],
	mounted() {
		this.initTooltip();
	},
	watch: {
		editable() {
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
</style>
