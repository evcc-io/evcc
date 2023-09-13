<template>
	<header class="d-flex justify-content-between align-items-center py-3">
		<h1 class="mb-1 pt-1 d-flex text-nowrap text-truncate">
			<router-link class="evcc-default-text" to="/">
				<shopicon-regular-home size="s" class="home"></shopicon-regular-home>
			</router-link>
			<div size="s" class="mx-2 flex-grow-0 flex-shrink-0 fw-normal">/</div>
			<span class="text-truncate">{{ title }}</span>
		</h1>
		<TopNavigation />
	</header>
</template>

<script>
import "@h2d2/shopicons/es/regular/home";
import TopNavigation from "./TopNavigation.vue";

export default {
	name: "TopHeader",
	components: {
		TopNavigation,
	},
	props: {
		title: String,
	},
};
</script>

<style scoped>
.home {
	height: 22px;
	width: 22px;
	position: relative;
	top: -3px;
}
</style>
