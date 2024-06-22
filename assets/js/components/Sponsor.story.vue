<script setup>
import Sponsor from "./Sponsor.vue";
</script>

<template>
	<Story :layout="{ type: 'grid', width: 600 }">
		<Variant title="no sponsor">
			<Sponsor />
		</Variant>
		<Variant title="trial">
			<Sponsor sponsor="trial" />
		</Variant>
		<Variant title="sponsor">
			<Sponsor sponsor="naltatis" />
		</Variant>
		<Variant title="victron device">
			<Sponsor sponsor="victron" />
		</Variant>
	</Story>
</template>
