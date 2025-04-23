<template>
	<label
		ref="tooltip"
		class="root position-relative d-block"
		role="button"
		data-testid="loadpoint-prio-label"
		data-bs-toggle="tooltip"
		data-bs-html="true"
		:title="tooltipTitle"
		:class="{ 'opacity-25': !isEditable }"
		:aria-label="isEditable ? $t('main.loadpointPrio.prioTooltip') : null"
	>
		<CustomSelect
			v-if="isEditable"
			:selected="currentPrio"
			:options="priorityOptions"
			data-testid="prioritySelect"
			@change="setPriority($event.target.value)"
		>
			<PriorityIcon :prio="currentPrio" />
		</CustomSelect>
		<PriorityIcon v-else :prio="currentPrio" />
	</label>
</template>

<script>
import Tooltip from "bootstrap/js/dist/tooltip";
import CustomSelect from "../../Helper/CustomSelect.vue";
import PriorityIcon from "./PriorityIcon.vue";

export default {
	name: "PrioritySelect",
	components: {
		CustomSelect,
		PriorityIcon,
	},
	props: {
		effectivePriority: Number,
		priority: Number,
		editable: {
			type: Boolean,
			default: undefined,
		},
	},
	emits: ["priority-updated"],
	data() {
		return {
			tooltip: null,
			currentPrio: null,
		};
	},
	computed: {
		isEditable() {
			if (this.editable !== undefined) return this.editable;
			return this.effectivePriority === this.priority;
		},
		priorityOptions() {
			return [
				{ value: -3, name: this.$t("main.loadpointPrio.veryLow") },
				{ value: -2, name: this.$t("main.loadpointPrio.lower") },
				{ value: -1, name: this.$t("main.loadpointPrio.low") },
				{ value: 0, name: this.$t("main.loadpointPrio.normal") },
				{ value: 1, name: this.$t("main.loadpointPrio.high") },
				{ value: 2, name: this.$t("main.loadpointPrio.higher") },
				{ value: 3, name: this.$t("main.loadpointPrio.veryHigh") },
			];
		},
		tooltipTitle() {
			if (this.isEditable) {
				return ""; // Kein Tooltip wenn nicht editierbar
			}
			let title = ""; //`${this.$t("main.loadpointPrio.loadpoint")}: <span class='font-monospace'>${this.currentPrio}</span>`;
			title += `<div class="mt-1">${this.$t("main.loadpointPrio.prioTooltip")}</div>`;
			return `<div class="text-start">${title}</div>`;
		},
	},
	watch: {
		tooltipTitle() {
			this.initTooltip();
		},
		priority: "updatePrio",
		effectivePriority: "updatePrio",
	},
	mounted() {
		this.updatePrio();
		this.initTooltip();
	},
	methods: {
		updatePrio() {
			const prio =
				this.effectivePriority != null && this.effectivePriority !== this.priority
					? this.effectivePriority
					: this.priority;
			this.currentPrio = prio;
		},
		initTooltip() {
			this.$nextTick(() => {
				this.tooltip?.dispose();
				if (this.$refs.tooltip) {
					this.tooltip = new Tooltip(this.$refs.tooltip);
				}
			});
		},
		setPriority(value) {
			this.$emit("priority-updated", value);
			this.currentPrio = Number(value);
			this.initTooltip();
		},
	},
};
</script>
