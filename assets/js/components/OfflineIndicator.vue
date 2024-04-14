<template>
	<div
		class="fixed-bottom alert alert-secondary d-flex justify-content-center align-items-center mb-0 rounded-0"
		role="alert"
	>
		<shopicon-regular-cloud class="me-2"></shopicon-regular-cloud>
		{{ $t("offline.message") }}
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/cloud";

export default {
	name: "OfflineIndicator",
};
</script>
<style>
.fixed-bottom {
	/* $zindex-toast https://getbootstrap.com/docs/5.3/layout/z-index/ */
	z-index: 1090 !important;
}
</style>
