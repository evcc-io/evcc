<template>
	<div class="root border d-flex gap-1" role="group">
		<slot></slot>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "IconSelectGroup",
});
</script>

<style scoped>
.root {
	border-radius: 20px;
	padding: 4px;
}
</style>
