<template>
	<div class="alert alert-warning my-4">
		{{ $t("config.sponsor.tokenRequired") }}
		<a href="#" role="button" class="text-warning" @click.prevent="openModal">
			{{ $t("config.sponsor.tokenRequiredLearnMore") }}
		</a>
	</div>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";

export default {
	name: "SponsorTokenRequired",
	methods: {
		openModal() {
			Modal.getOrCreateInstance(document.getElementById("sponsorModal")).show();
		},
	},
};
</script>
