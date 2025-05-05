<template>
	<div class="mode-group border d-inline-flex" role="group" data-testid="mode">
		<button
			v-for="m in modes"
			:key="m"
			type="button"
			class="btn flex-grow-1 flex-shrink-1 text-truncate-xs-only"
			:class="{ active: isActive(m) }"
			tabindex="0"
			@click="setTargetMode(m)"
		>
			{{ label(m) }}
		</button>
	</div>
</template>

<script lang="ts">
import { CHARGE_MODE } from "@/types/evcc";
import { defineComponent } from "vue";

const { OFF, PV, MINPV, NOW } = CHARGE_MODE;

export default defineComponent({
	name: "Mode",
	props: {
		mode: String,
		pvPossible: Boolean,
		hasSmartCost: Boolean,
	},
	emits: ["updated"],

	computed: {
		modes(): CHARGE_MODE[] {
			if (this.pvPossible) {
				return [OFF, PV, MINPV, NOW];
			}
			if (this.hasSmartCost) {
				return [OFF, PV, NOW];
			}
			return [OFF, NOW];
		},
	},
	methods: {
		label(mode: CHARGE_MODE) {
			// rename pv mode to smart for non-pv and dynamic tariffs scenarios
			// TODO: rollout smart name for everyting later
			if (mode === PV && !this.pvPossible && this.hasSmartCost) {
				return this.$t("main.mode.smart");
			}
			return this.$t(`main.mode.${mode}`);
		},
		isActive(mode: CHARGE_MODE) {
			return this.mode === mode;
		},
		setTargetMode(mode: CHARGE_MODE) {
			this.$emit("updated", mode);
		},
	},
});
</script>

<style scoped>
.mode-group {
	border: 2px solid var(--evcc-default-text);
	border-radius: 20px;
	padding: 4px;
	min-width: 255px;
}

.btn {
	/* equal width buttons */
	flex-basis: 0;
	white-space: nowrap;
	border-radius: 18px;
	padding: 0.1em 0.8em;
	color: var(--evcc-default-text);
	border: none;
}
@media (max-width: 576px) {
	.btn {
		padding: 0.1em 0.2em;
	}
}

.btn:hover {
	color: var(--evcc-gray);
}
.btn:focus {
	outline: var(--bs-focus-ring-width) solid var(--bs-focus-ring-color);
	outline-width: var(--bs-focus-ring-width);
}
.btn.active {
	color: var(--evcc-background);
	background: var(--evcc-default-text);
}
.btn-group {
	border-radius: 16px;
}
</style>
