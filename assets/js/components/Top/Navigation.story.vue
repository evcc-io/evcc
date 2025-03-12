<script setup>
import Navigation from "./Navigation.vue";
</script>

<template>
	<Story title="Top Navigation">
		<Variant title="standard">
			<Navigation />
		</Variant>
		<Variant title="vehicle logins">
			<Navigation
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
			<Navigation
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
		<Variant title="token expires">
			<Navigation :sponsor="{ expiresSoon: true }" />
		</Variant>
	</Story>
</template>
