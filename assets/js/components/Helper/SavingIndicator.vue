<template>
	<div class="small">
		<span v-if="state === 'saving'">{{ $t("config.general.saving") }}</span>
		<span v-else-if="state === 'saved'" class="text-primary">
			{{ $t("config.general.saved") }}
		</span>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";

export type SavingState = "saving" | "saved" | null;

export default defineComponent({
	name: "SavingIndicator",
	props: {
		state: { type: String as PropType<SavingState>, default: null },
	},
});
</script>
