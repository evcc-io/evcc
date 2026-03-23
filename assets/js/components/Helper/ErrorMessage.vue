<template>
	<p v-if="error" class="text-danger text-break">
		<strong>{{ $t("config.general.error") }}:</strong> {{ error }}
	</p>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	name: "ErrorMessage",
	props: {
		error: {
			type: String as PropType<string | null>,
			default: null,
		},
	},
});
</script>
