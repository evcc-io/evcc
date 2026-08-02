<template>
	<div v-if="visible.length" class="steps">
		<div v-for="(step, i) in visible" :key="i" class="step">
			<p v-if="step.reasoning" class="reasoning mb-0">{{ step.reasoning }}</p>
			<p v-for="(call, j) in step.calls" :key="j" class="call mb-0">
				{{ call.name }}<span v-if="call.arguments">{{ call.arguments }}</span>
			</p>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { cleanReasoning } from "@/utils/reasoning";
import type { AssistantStep } from "@/types/evcc";

export default defineComponent({
	name: "AssistantSteps",
	props: {
		steps: { type: Array as PropType<AssistantStep[]>, default: () => [] },
	},
	computed: {
		// a step whose reasoning was nothing but tool call markup has nothing left to show
		visible(): AssistantStep[] {
			return this.steps
				.map((step) => ({ ...step, reasoning: cleanReasoning(step.reasoning || "") }))
				.filter((step) => step.reasoning || step.calls?.length);
		},
	},
});
</script>

<style scoped>
.steps {
	color: var(--evcc-gray);
	font-size: 0.875rem;
}
.step {
	margin-bottom: 0.5rem;
}
.reasoning {
	white-space: pre-wrap;
	overflow-wrap: anywhere;
}
.call {
	font-family: var(--bs-font-monospace);
	overflow-wrap: anywhere;
}
.call span {
	opacity: 0.7;
}
</style>
