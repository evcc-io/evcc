<template>
	<div class="card">
		<div class="card-header card-header-with-link">
			<h3 class="mb-0">Hausinstallation</h3>
			<slot name="action"></slot>
		</div>
		<div class="card-body">
			<slot name="content"></slot>
		</div>
	</div>
</template>

<script>
export default {
	name: "Card",
	props: {
		title: String,
	},
};
</script>

<style scoped>
.card-header-with-link {
	display: flex;
	justify-content: space-between;
	align-items: baseline;
}
</style>
