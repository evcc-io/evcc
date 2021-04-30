<template>
	<a
		href="https://github.com/sponsors/andig"
		target="_blank"
		class="btn btn-link text-decoration-none link-dark text-nowrap"
	>
		<div v-if="supporter">
			<span>Supporter</span>
			<fa-icon icon="star" class="icon ms-1"></fa-icon>
		</div>
		<div v-else>
			<span class="d-none d-sm-inline">Projekt </span>unterstützen
			<fa-icon icon="heart" class="icon ms-1"></fa-icon>
		</div>
	</a>
</template>

<script>
export default {
	name: "Supporter",
	props: {
		supporter: Boolean,
	},
};
</script>

<style scoped>
.icon {
	color: #0fdd42;
}
</style>
