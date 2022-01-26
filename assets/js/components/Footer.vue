<template>
	<footer class="footer">
		<div class="container py-3">
			<div class="d-flex justify-content-between">
				<Version v-bind="version" />
				<!-- Please don't mess with the sponsor status. You risk loosing your mojo. -->
				<Savings v-bind="savings" :sponsor="sponsor" />
			</div>
		</div>
	</footer>
</template>

<script>
import Version from "./Version";
import Savings from "./Savings";

export default {
	name: "Footer",
	components: { Version, Savings },
	props: {
		version: Object,
		sponsor: String,
		savings: Object,
	},
};
</script>

<style scoped>
.footer {
	color: var(--bs-white);
}
</style>
