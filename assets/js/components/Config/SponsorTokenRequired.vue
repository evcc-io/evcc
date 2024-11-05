<template>
	<div class="alert alert-success my-4">
		{{ $t("config.sponsor.tokenRequired") }}
		<a href="#" role="button" class="text-success" @click.prevent="openModal">
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
