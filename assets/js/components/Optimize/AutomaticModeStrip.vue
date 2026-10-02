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
import { automaticLevelActions, automaticLevelOptions } from "./automaticLevels";

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
			return automaticLevelOptions(this.$t, this.isSponsor);
		},
		description(): string {
			const actions = automaticLevelActions(this.automatic);
			if (actions.length === 0) {
				return this.$t("config.optimizer.automaticOff");
			}
			return actions.map((key) => this.$t(key)).join(", ") + ".";
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
