<template>
	<footer class="footer">
		<div class="container py-2">
			<div class="d-flex justify-content-between">
				<Version v-bind="version" />
				<!--
				<button
					class="btn btn-link p-0 flex-shrink-0"
					:class="{ 'text-evcc': $hiddenFeatures, 'text-muted': !$hiddenFeatures }"
					@click="toggleHiddenFeatures"
				>
					<shopicon-filled-testtube size="s"></shopicon-filled-testtube>
				</button>-->
				<!-- Please don't mess with the sponsor status. You risk loosing your mojo. -->
				<Savings v-bind="savings" :sponsor="sponsor" />
			</div>
		</div>
	</footer>
</template>

<script>
import "@h2d2/shopicons/es/filled/testtube";

import Version from "./Version.vue";
import Savings from "./Savings.vue";

export default {
	name: "Footer",
	components: { Version, Savings },
	props: {
		version: Object,
		sponsor: String,
		savings: Object,
	},
	methods: {
		toggleHiddenFeatures() {
			const dialog = `🧪 Experimentelle UI-Elemente ${
				this.$hiddenFeatures ? "deaktivieren" : "aktivieren"
			}`;
			if (window.confirm(dialog)) {
				window.localStorage["hidden_features"] = !this.$hiddenFeatures;
				window.location.reload();
			}
		},
	},
};
</script>

<style scoped></style>
