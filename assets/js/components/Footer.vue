<template>
	<footer class="py-3 py-md-5 mt-3 mt-md-5 border-top">
		<div class="container">
			<div class="row">
				<div class="col-6">
					<Version v-bind="version" />
				</div>
				<div class="col-6 text-right">
					<small class="text-muted">
						<a href="https://github.com/sponsors/andig" target="_blank">
							<fa-icon icon="heart"></fa-icon>
							Projekt unterstützen
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
