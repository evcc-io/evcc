<template>
	<footer class="container">
		<div class="py-3 py-md-5 mt-3 mt-md-5 border-top">
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
