<template>
	<footer class="footer" data-testid="footer">
		<div class="container py-2">
			<div class="d-flex justify-content-between">
				<Version v-bind="version" />
				<Savings v-bind="savings" />
			</div>
		</div>
	</footer>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/filled/testtube";

import Version from "./Version.vue";
import Savings from "../Savings/Savings.vue";
import { defineComponent } from "vue";

export default defineComponent({
	name: "Footer",
	components: { Version, Savings },
	props: {
		version: Object,
		savings: Object,
	},
});
</script>
