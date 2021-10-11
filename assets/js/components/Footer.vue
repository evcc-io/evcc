<template>
	<footer class="container">
		<div class="py-3 py-md-5 mt-3 mt-md-5 border-top position-relative">
			<div class="d-flex justify-content-between">
				<Version v-bind="version" />
				<!-- Please don't mess with the sponsor status. You risk loosing your mojo. -->
				<Sponsor :sponsor="sponsor" />
			</div>
			<Savings v-bind="savings" :sponsor="sponsor" />
		</div>
	</footer>
</template>

<script>
import Savings from "./Savings";
import Version from "./Version";
import Sponsor from "./Sponsor";

export default {
	name: "Footer",
	components: { Savings, Version, Sponsor },
	props: {
		version: Object,
		sponsor: String,
		savings: Object,
	},
};
</script>
