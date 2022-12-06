<script setup>
import TopNavigation from "./TopNavigation.vue";
</script>

<template>
	<Story title="Top Navigation">
		<Variant title="standard">
			<TopNavigation />
		</Variant>
		<Variant title="vehicle logins">
			<TopNavigation
				:vehicleLogins="{
					['Mercedes EQS']: {
						authenticated: true,
						uri: 'https://login-provider-a.test/',
					},
					['Nissan Leaf Pro']: {
						authenticated: true,
						uri: 'https://login-provider-b.test/',
					},
				}"
			/>
		</Variant>
		<Variant title="pending vehicle logins">
			<TopNavigation
				:vehicleLogins="{
					['Mercedes EQS']: {
						authenticated: true,
						uri: 'https://login-provider-a.test/',
					},
					['Nissan Leaf Pro']: {
						authenticated: false,
						uri: 'https://login-provider-b.test/',
					},
				}"
			/>
		</Variant>
	</Story>
</template>
