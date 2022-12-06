<script setup>
import Sponsor from "./Sponsor.vue";
</script>

<template>
	<Story :layout="{ type: 'grid', width: 600 }">
		<Variant title="no sponsor">
			<Sponsor />
		</Variant>
		<Variant title="sponsor">
			<Sponsor sponsor="naltatis" />
		</Variant>
	</Story>
</template>
