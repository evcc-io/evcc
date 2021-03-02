<template>
	<footer class="py-3 py-md-5 mt-3 mt-md-5 border-top">
		<div class="container">
			<div class="row">
				<div class="col-6">
					<p class="text-muted">
						<Version v-bind="version" />
					</p>
				</div>
				<div class="col-6 text-right">
					<small class="text-muted">
						<a href= "https://github.com/mark-sch/evcc" target="_blank">&copy; 2020 andig/evcc, Think5 GmbH et al.</a>
					</small>
				</div>
			</div>
		</div>
	</footer>
</template>

<script>
import Version from "./Version";

export default {
	name: "Footer",
	components: { Version },
	props: {
		version: Object,
	},
};
</script>

<style scoped>
.icon {
	color: #0fdd42;
}
.text-black a {
	color: #18191a;
}
</style>
