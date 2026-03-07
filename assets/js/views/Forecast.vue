<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader :title="$t('forecast.modalTitle')" />
		<div class="row">
			<main class="col-12">
				<p class="text-gray mt-4 text-center">{{ $t("tabBar.comingSoon") }}</p>
			</main>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import Header from "../components/Top/Header.vue";

export default defineComponent({
	name: "Forecast",
	components: {
		TopHeader: Header,
	},
	head() {
		return { title: this.$t("forecast.modalTitle") };
	},
});
</script>
