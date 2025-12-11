<template>
	<div class="dropdown">
		<button
			class="d-flex align-items-center gap-2 btn btn-secondary dropdown-toggle"
			type="button"
			data-bs-toggle="dropdown"
			aria-expanded="false"
		>
			<slot></slot>
		</button>
		<ul class="dropdown-menu">
			<li v-for="a in actions">
				<a class="dropdown-item" @click="$emit('click', a.value)">
					{{ a.name }}
				</a>
			</li>
		</ul>
	</div>
</template>

<script lang="ts">
import type { SelectActionOption } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	name: "CustomSelect",
	props: {
		actions: { type: Array as PropType<SelectActionOption<number | string>[]> },
	},
	emits: ["click"],
});
</script>
