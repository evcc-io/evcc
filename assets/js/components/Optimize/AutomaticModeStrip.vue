<template>
	<div
		class="strip d-flex flex-column flex-sm-row align-items-sm-baseline gap-2 gap-sm-3 px-3 px-sm-4 py-3"
		:class="{ 'strip--on': automatic !== OPTIMIZER_AUTOMATIC.OFF }"
		data-testid="optimizer-automatic-strip"
	>
		<div class="d-flex align-items-center gap-2 flex-shrink-0">
			<label class="fw-bold text-nowrap" for="optimizerAutomaticStrip">
				{{ $t("config.optimizer.automatic") }}
			</label>
			<SelectGroup
				id="optimizerAutomaticStrip"
				transparent
				:model-value="automatic"
				:options="levelOptions"
				:aria-label="$t('config.optimizer.automatic')"
				@update:model-value="$emit('change', $event)"
			/>
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
import { defineComponent, type PropType } from "vue";
import SelectGroup from "../Helper/SelectGroup.vue";
import { OPTIMIZER_AUTOMATIC, type SelectOption } from "@/types/evcc";

export default defineComponent({
	name: "AutomaticModeStrip",
	components: { SelectGroup },
	props: {
		automatic: {
			type: String as PropType<OPTIMIZER_AUTOMATIC>,
			default: OPTIMIZER_AUTOMATIC.OFF,
		},
		isSponsor: Boolean,
	},
	emits: ["change", "learn-more"],
	data() {
		return { OPTIMIZER_AUTOMATIC };
	},
	computed: {
		levelOptions(): SelectOption<string>[] {
			return Object.values(OPTIMIZER_AUTOMATIC).map((value) => ({
				value,
				name: this.$t(`config.optimizer.automaticLevel.${value}`),
				disabled: !this.isSponsor,
			}));
		},
		description(): string {
			switch (this.automatic) {
				case OPTIMIZER_AUTOMATIC.BATTERY:
					return this.$t("config.optimizer.automaticBattery") + ".";
				case OPTIMIZER_AUTOMATIC.FULL:
					return (
						[
							"config.optimizer.automaticBattery",
							"config.optimizer.automaticCharging",
							"config.optimizer.automaticLimits",
							"config.optimizer.automaticPlans",
						]
							.map((key) => this.$t(key))
							.join(", ") + "."
					);
				default:
					return this.$t("config.optimizer.automaticOff");
			}
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
