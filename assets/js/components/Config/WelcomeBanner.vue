<template>
	<div class="alert alert-success" data-testid="welcome-banner">
		<h5 class="alert-heading">{{ $t("config.main.welcomeBannerTitle") }}</h5>
		<Markdown :markdown="$t('config.main.welcomeBannerText')" class="mb-0" />
	</div>
</template>

<script>
import Markdown from "./Markdown.vue";

export default {
	name: "WelcomeBanner",
	components: { Markdown },
};
</script>
