<template>
	<div class="strip d-flex align-items-start gap-2 px-4 py-3" :class="{ 'strip--on': enabled }">
		<!-- zero-width space gives the dot a text line to center in, aligned with the first line -->
		<StatusIndicator :variant="enabled ? 'success' : 'muted'" class="flex-shrink-0">
			&#8203;
		</StatusIndicator>
		<div class="d-flex flex-wrap align-items-baseline column-gap-3 row-gap-1 flex-grow-1">
			<i18n-t
				:keypath="`config.optimizer.automaticLevel.${automatic}.status`"
				tag="div"
				scope="global"
				class="flex-grow-1"
			>
				<template #automatic>
					<span class="fw-bold">{{ $t("config.optimizer.automatic") }}</span>
				</template>
			</i18n-t>
			<a href="#" class="fw-bold" @click.prevent="$emit('change')">
				{{ $t("config.general.change") }}
			</a>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import StatusIndicator from "../Config/StatusIndicator.vue";
import { OPTIMIZER_AUTOMATIC } from "@/types/evcc";

export default defineComponent({
	name: "AutomaticModeStrip",
	components: { StatusIndicator },
	props: {
		automatic: {
			type: String as PropType<OPTIMIZER_AUTOMATIC>,
			default: OPTIMIZER_AUTOMATIC.OFF,
		},
	},
	emits: ["change"],
	computed: {
		enabled(): boolean {
			return this.automatic !== OPTIMIZER_AUTOMATIC.OFF;
		},
	},
});
</script>

<style scoped>
/* full-bleed strip flush with the card bottom: square edge-to-edge card below sm, rounded above */
.strip {
	margin: 1rem -1.5rem -1rem;
	color: var(--evcc-gray);
}
@media (min-width: 576px) {
	.strip {
		margin: 1.5rem -1.5rem -1.5rem;
		border-radius: 0 0 1rem 1rem;
	}
}
.strip--on {
	background: var(--evcc-success-bg);
	color: var(--evcc-success-text);
}
.strip a {
	color: inherit;
}
</style>
