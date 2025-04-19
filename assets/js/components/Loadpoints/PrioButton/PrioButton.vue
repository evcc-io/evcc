<template>
	<button
		v-if="!editable"
		ref="tooltip"
		type="button"
		class="btn btn-sm btn-outline-secondary position-relative border-0 pt-0"
		data-testid="loadpoint-prio-button"
		:class="{ 'opacity-25': !editable }"
		data-bs-toggle="tooltip"
		data-bs-html="true"
		:title="tooltipTitle"
		:aria-label="editable ? $t('main.loadpointPrio.prioTooltip') : null"
	>
		<component :is="icon" :class="`icon icon--${size}`"></component>
	</button>
	<button
		v-if="editable"
		ref="tooltip"
		type="button"
		class="btn btn-sm btn-outline-secondary position-relative border-0 pt-0"
		data-testid="loadpoint-prio-button"
		:class="{ 'opacity-25': !editable }"
		data-bs-toggle="tooltip"
		data-bs-html="true"
		:title="tooltipTitle"
		:aria-label="editable ? $t('main.loadpointPrio.prioTooltip') : null"
	>
		<CustomSelect
			v-if="editable"
			:selected="prio"
			:options="priorityOptions"
			data-testid="prioritySelect"
			@change="setPriority($event.target.value)"
		>
			<component :is="icon" :class="`icon icon--${size}`"></component>
		</CustomSelect>
	</button>
</template>

<script>
import Tooltip from "bootstrap/js/dist/tooltip";
import CustomSelect from "../../Helper/CustomSelect.vue";

import _zero from "./zero.vue";
import _minus1 from "./minus-1.vue";
import _minus2 from "./minus-2.vue";
import _minus3 from "./minus-3.vue";
import _plus1 from "./plus-1.vue";
import _plus2 from "./plus-2.vue";
import _plus3 from "./plus-3.vue";

export default {
	name: "PrioButton",
	components: {
		CustomSelect,
	},
	props: {
		prio: { type: Number },
		size: { type: String, default: "sm" },
		editable: Boolean,
	},
	emits: ["update:prio"],
	data() {
		return {
			tooltip: null,
		};
	},
	computed: {
		icon() {
			// Shortened logic with direct return of components
			if (this.prio === 0) return _zero;
			if (this.prio === 1) return _plus1;
			if (this.prio === 2) return _plus2;
			if (this.prio === 3) return _plus3;
			if (this.prio === -1) return _minus1;
			if (this.prio === -2) return _minus2;
			if (this.prio === -3) return _minus3;

			// For prios beyond the range of -3 to 3
			return this.prio > 3 ? _plus3 : _minus3;
		},
		priorityOptions() {
			return [
				{ value: -3, name: this.$t("main.loadpointPrio.veryLow") },
				{ value: -2, name: this.$t("main.loadpointPrio.low") },
				{ value: -1, name: this.$t("main.loadpointPrio.slightlyLow") },
				{ value: 0, name: this.$t("main.loadpointPrio.neutral") },
				{ value: 1, name: this.$t("main.loadpointPrio.slightlyHigh") },
				{ value: 2, name: this.$t("main.loadpointPrio.high") },
				{ value: 3, name: this.$t("main.loadpointPrio.veryHigh") },
			];
		},
		tooltipTitle() {
			let title = `${this.$t("main.loadpointPrio.loadpoint")}: <span class='font-monospace'>${this.prio}</span>`;
			if (!this.editable) {
				title += `<div class="mt-1">${this.$t("main.loadpointPrio.prioTooltip")}</div>`;
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
		initTooltip() {
			this.$nextTick(() => {
				this.tooltip?.dispose();
				if (this.$refs.tooltip) {
					this.tooltip = new Tooltip(this.$refs.tooltip);
				}
			});
		},
		setPriority(value) {
			this.$emit("update:prio", value);
			this.initTooltip();
		},
	},
};
</script>

<style scoped>
.icon {
	display: block;
	width: 24px;
	height: 24px;
}
.icon--m {
	width: 32px;
	height: 32px;
}
.icon--l {
	width: 48px;
	height: 48px;
}
.icon--xl {
	width: 64px;
	height: 64px;
}
</style>
