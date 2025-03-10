<script setup>
import SponsorTokenExpires from "./SponsorTokenExpires.vue";
</script>

<template>
	<Story title="Sponsor Token Expires">
		<Variant title="some day">
			<SponsorTokenExpires :sponsorSoon="true" expiresAt="2055-01-01T00:00:00Z" />
		</Variant>
		<Variant title="empty">
			<SponsorTokenExpires />
		</Variant>
	</Story>
</template>
