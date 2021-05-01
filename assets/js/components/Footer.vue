<template>
	<footer class="container">
		<div class="py-3 py-md-5 mt-3 mt-md-5 border-top">
			<div class="d-flex justify-content-between">
				<Version v-bind="version" />
				<!-- Please don't mess with the supporter status. You risk loosing your mojo. -->
				<Supporter :supporter="supporter" />
			</div>
		</div>
	</footer>
</template>

<script>
import Version from "./Version";
import Supporter from "./Supporter";

export default {
	name: "Footer",
	components: { Version, Supporter },
	props: {
		version: Object,
		supporter: Boolean,
	},
};
</script>
