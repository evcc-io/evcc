<template>
	<footer class="py-3 py-md-5 mt-3 mt-md-5 border-top">
		<div class="container">
			<div class="row">
				<div class="col-6">
					<Version v-bind="version" />
				</div>
				<div class="col-6 text-right">
					<small class="text-black">
						<a href="https://github.com/sponsors/andig" target="_blank" class="link">
							<fa-icon icon="heart" class="icon mr-1"></fa-icon
							><span class="d-none d-sm-inline">Projekt </span>unterstützen
						</a>
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
