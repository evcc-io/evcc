<template>
	<details v-if="steps.length" class="steps">
		<summary>{{ $t("assistant.steps", { count: steps.length }) }}</summary>
		<div v-for="(step, i) in steps" :key="i" class="step">
			<p v-if="step.reasoning" class="reasoning mb-1">{{ step.reasoning }}</p>
			<p v-for="(call, j) in step.calls" :key="j" class="call mb-1">
				{{ call.name }}<span v-if="call.arguments">{{ call.arguments }}</span>
			</p>
		</div>
	</details>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import type { AssistantStep } from "@/types/evcc";

export default defineComponent({
	name: "AssistantSteps",
	props: {
		steps: { type: Array as PropType<AssistantStep[]>, default: () => [] },
	},
});
</script>

<style scoped>
.steps {
	margin-bottom: 0.5rem;
	color: var(--evcc-gray);
	font-size: 0.875rem;
}
summary {
	cursor: pointer;
}
.step {
	margin-top: 0.5rem;
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
