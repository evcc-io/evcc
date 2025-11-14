<template>
	<JsonModal
		id="networkModal"
		:title="$t('config.network.title')"
		endpoint="/config/network"
		state-key="network"
		:transform-write-values="transformWriteValues"
		disable-remove
		data-testid="network-modal"
		@changed="$emit('changed')"
	>
		<template #default="{ values }">
			<FormRow
				id="networkHost"
				:label="$t('config.network.labelHost')"
				:help="$t('config.network.descriptionHost')"
				example="evcc.local"
				optional
			>
				<input
					id="networkHost"
					v-model="values.host"
					class="form-control"
					spellcheck="false"
				/>
			</FormRow>
			<FormRow
				id="networkPort"
				:label="$t('config.network.labelPort')"
				:help="$t('config.network.descriptionPort')"
				example="7070"
			>
				<input
					id="networkPort"
					v-model="values.port"
					class="form-control w-50 me-2 w-50"
					type="number"
					required
				/>
			</FormRow>

			<FormRow
				id="networkInternalUrl"
				:label="$t('config.network.labelInternalUrl')"
				:help="$t('config.network.descriptionInternalUrl')"
			>
				<input
					id="networkInternalUrl"
					v-model="values.internalUrl"
					class="form-control"
					type="text"
					readonly
					tabindex="-1"
				/>
			</FormRow>

			<FormRow
				id="networkExternalUrl"
				:label="$t('config.network.labelExternalUrl')"
				:help="$t('config.network.descriptionExternalUrl')"
				example="https://foo.local:220"
				optional
			>
				<input
					id="networkExternalUrl"
					v-model="values.externalUrl"
					class="form-control"
					type="text"
					inputmode="url"
					autocomplete="off"
					spellcheck="false"
				/>
			</FormRow>
		</template>
	</JsonModal>
</template>

<script>
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";

export default {
	name: "NetworkModal",
	components: { FormRow, JsonModal },
	emits: ["changed"],
	methods: {
		transformWriteValues(values) {
			const payload = { ...values };
			delete payload.internalUrl;

			return payload;
		},
	},
};
</script>
