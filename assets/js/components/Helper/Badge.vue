<template>
	<span class="badge rounded-pill" :class="`badge--${variant}`">
		<slot />
	</span>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";

type Variant = "success" | "muted";

export default defineComponent({
	name: "Badge",
	props: {
		variant: { type: String as PropType<Variant>, default: "success" },
	},
});
</script>

<style scoped>
.badge {
	--bs-badge-font-size: 0.75rem;
	--bs-badge-padding-y: 0.25em;
	font-weight: 700;
}
.badge--success {
	background: var(--evcc-green);
	color: var(--evcc-darkest-green);
}
.badge--muted {
	background: var(--evcc-gray-10);
	color: var(--evcc-gray);
}
</style>
