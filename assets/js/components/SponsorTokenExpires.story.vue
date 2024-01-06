<script setup>
import SponsorTokenExpires from "./SponsorTokenExpires.vue";
</script>

<template>
	<Story title="Sponsor Token Expires">
		<Variant title="in 9 days">
			<SponsorTokenExpires :sponsorTokenExpires="768423" />
		</Variant>
		<Variant title="empty">
			<SponsorTokenExpires />
		</Variant>
	</Story>
</template>
