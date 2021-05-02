<template>
	<footer class="container">
		<div class="py-3 py-md-5 mt-3 mt-md-5 border-top">
			<div class="d-flex justify-content-between">
				<Version v-bind="version" />
				<!-- Please don't mess with the sponsor status. You risk loosing your mojo. -->
				<Sponsor :sponsor="sponsor" />
			</div>
		</div>
	</footer>
</template>

<script>
import Version from "./Version";
import Sponsor from "./Sponsor";

export default {
	name: "Footer",
	components: { Version, Sponsor },
	props: {
		version: Object,
		sponsor: String,
	},
};
</script>
