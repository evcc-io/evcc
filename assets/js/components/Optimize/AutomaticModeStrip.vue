<template>
	<div
		class="strip d-flex flex-column flex-sm-row align-items-sm-baseline gap-2 gap-sm-3 px-3 px-sm-4 py-3"
		:class="{ 'strip--on': automatic }"
		data-testid="optimizer-automatic-strip"
	>
		<div class="form-check form-switch m-0 flex-shrink-0">
			<input
				id="optimizerAutomaticStrip"
				:checked="automatic"
				class="form-check-input"
				type="checkbox"
				role="switch"
				:disabled="!isSponsor"
				@change="change"
			/>
			<label class="form-check-label fw-bold text-nowrap" for="optimizerAutomaticStrip">
				{{ $t("config.optimizer.automatic") }}
			</label>
		</div>
		<div class="description">
			{{ description }}
			<a href="#" class="fw-bold" @click.prevent="$emit('learn-more')">
				{{ $t("config.general.learnMore") }}
			</a>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "AutomaticModeStrip",
	props: {
		automatic: Boolean,
		isSponsor: Boolean,
	},
	emits: ["change", "learn-more"],
	computed: {
		description(): string {
			const bullets = [
				"config.optimizer.automaticCharging",
				"config.optimizer.automaticBattery",
				"config.optimizer.automaticLimits",
				"config.optimizer.automaticPlans",
			];
			return bullets.map((key) => this.$t(key)).join(", ") + ".";
		},
	},
	methods: {
		change(e: Event) {
			this.$emit("change", (e.target as HTMLInputElement).checked);
		},
	},
});
</script>

<style scoped>
/* full-bleed strip flush with the card's rounded bottom (card padding is p-3/p-sm-4) */
.strip {
	margin: 1rem -1rem -1rem;
	border-radius: 0 0 1rem 1rem;
	color: var(--evcc-gray);
}
@media (min-width: 576px) {
	.strip {
		margin: 1.5rem -1.5rem -1.5rem;
	}
}
.strip--on {
	background: var(--evcc-success-bg);
	color: var(--evcc-success-text);
}
.strip--on .description {
	color: var(--evcc-success-text-secondary);
}
.strip a {
	color: inherit;
}
</style>
