<template>
	<div v-if="compact" class="mt-4">
		<a href="#" role="button" class="text-danger" @click.prevent="open">
			{{ $t("config.sponsor.tokenRequiredShort") }}
		</a>
	</div>
	<div v-else class="alert alert-warning my-4">
		{{ $t(feature ? "config.sponsor.tokenRequiredFeature" : "config.sponsor.tokenRequired") }}
		<a href="#" role="button" class="text-warning" @click.prevent="open">
			{{ $t("config.sponsor.tokenRequiredLearnMore") }}
		</a>
	</div>
</template>

<script>
import { openModal } from "@/configModal";

export default {
	name: "SponsorTokenRequired",
	props: {
		compact: Boolean,
		feature: Boolean,
	},
	methods: {
		open() {
			openModal("sponsor");
		},
	},
};
</script>
