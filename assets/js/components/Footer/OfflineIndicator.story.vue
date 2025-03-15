<script setup>
import OfflineIndicator from "./OfflineIndicator.vue";
</script>

<template>
	<Story>
		<Variant title="offline">
			<OfflineIndicator offline />
		</Variant>
		<Variant title="fatal">
			<OfflineIndicator
				:fatal="{
					error: `cannot create charger 'wallbox_cphl': cannot create charger type 'template': cannot create
charger type 'hardybarth-ecbl: Post http://192.168.2.219/api/v1/chargecontrols/1/mode: context deadline exceeded (Client.Timeout
exceeded while awaiting headers)`,
				}"
			/>
		</Variant>
	</Story>
</template>
