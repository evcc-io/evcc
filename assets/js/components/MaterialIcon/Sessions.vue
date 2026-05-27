<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M4 16q-.425 0-.712-.288T3 15t.288-.712T4 14h6q.425 0 .713.288T11 15t-.288.713T10 16zm0-4q-.425 0-.712-.288T3 11t.288-.712T4 10h10q.425 0 .713.288T15 11t-.288.713T14 12zm0-4q-.425 0-.712-.288T3 7t.288-.712T4 6h10q.425 0 .713.288T15 7t-.288.713T14 8zm12.35 10.575q-.2 0-.375-.062t-.325-.213l-2.15-2.15q-.275-.275-.287-.687t.287-.713q.275-.275.688-.288t.712.263l1.45 1.425l3.525-3.525q.3-.3.713-.287t.712.312q.275.3.288.7t-.288.7l-4.25 4.25q-.15.15-.325.213t-.375.062"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "SessionsIcon",
	mixins: [icon],
});
</script>
