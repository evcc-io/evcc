<template>
	<div class="root border d-flex gap-1" role="group">
		<slot></slot>
	</div>
</template>

<script>
export default {
	name: "IconSelectGroup",
};
</script>

<style scoped>
.root {
	border-radius: 20px;
	padding: 4px;
}
</style>
