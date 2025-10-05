<template>
	<JsonModal
		id="shmModal"
		:title="$t('config.shm.title')"
		:description="$t('config.shm.description')"
		docs="/docs/reference/configuration/hems"
		endpoint="/config/shm"
		state-key="shm"
		data-testid="shm-modal"
		disable-remove
		@changed="$emit('changed')"
	>
		<template #default="{ values }">
			<PropertyCollapsible>
				<template #advanced>
					<p>{{ $t("config.shm.descriptionIds") }}</p>
					<p>
						{{ $t("config.shm.descriptionIdPattern") }}<br />
						<code>F-AAAAAAAA-BBBBBBBBBBBB-00</code>
					</p>
					<p>
						{{ $t("config.shm.descriptionSempUrl") }}<br />
						<a :href="sempUrl" target="_blank" data-testid="semp-url">{{ sempUrl }}</a>
					</p>
					<FormRow
						id="shmVendorid"
						:label="$t('config.shm.labelVendorId')"
						:help="$t('config.shm.descriptionVendorId')"
						example="AAAAAAAA"
						optional
					>
						<input
							id="shmVendorid"
							v-model="values.vendorId"
							class="form-control"
							minlength="8"
							maxlength="8"
							pattern="[A-Fa-f0-9]{8}"
						/>
					</FormRow>
					<FormRow
						id="shmDeviceid"
						:label="$t('config.shm.labelDeviceId')"
						:help="$t('config.shm.descriptionDeviceId')"
						example="BBBBBBBBBBBB"
						optional
					>
						<input
							id="shmDeviceid"
							v-model="values.deviceId"
							class="form-control"
							minlength="12"
							maxlength="12"
							pattern="[A-Fa-f0-9]{12}"
						/> </FormRow
				></template>
			</PropertyCollapsible>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import FormRow from "./FormRow.vue";
import JsonModal from "./JsonModal.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";

export default {
	name: "ShmModal",
	components: { JsonModal, FormRow, PropertyCollapsible },
	emits: ["changed"],
	computed: {
		sempUrl() {
			// TOOD: combine with url-gen logic coming in https://github.com/evcc-io/evcc/pull/23093
			return `${window.location.protocol}//${window.location.hostname}:${window.location.port}/semp/`;
		},
	},
};
</script>
