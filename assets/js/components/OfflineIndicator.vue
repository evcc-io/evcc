<template>
	<div
		class="fixed-bottom alert alert-secondary d-flex justify-content-center align-items-center mb-0 rounded-0"
		role="alert"
	>
		<shopicon-regular-cloud class="me-2"></shopicon-regular-cloud>
		{{ $t("offline.message") }}
		<button class="btn btn-sm btn-link text-dark" @click="reload">
			{{ $t("offline.reload") }}
		</button>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/cloud";

export default {
	name: "OfflineIndicator",
	methods: {
		reload() {
			window.location.reload();
		},
	},
};
</script>
