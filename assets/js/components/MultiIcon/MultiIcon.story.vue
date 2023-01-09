<script setup>
import MultiIcon from "./MultiIcon.vue";
</script>

<template>
	<Story :layout="{ type: 'grid', width: 150 }">
		<Variant v-for="count in [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]" :key="count" :title="count">
			<MultiIcon :count="count" size="xl" />
		</Variant>
	</Story>
</template>
