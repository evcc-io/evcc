<template>
	<JsonModal
		id="networkModal"
		:title="$t('config.network.title')"
		endpoint="/config/network"
		state-key="network"
		disable-remove
		data-testid="network-modal"
		@changed="$emit('changed')"
	>
		<template v-slot:default="{ values }">
			<FormRow
				id="networkSchema"
				:label="$t('config.network.labelSchema')"
				:help="$t('config.network.descriptionSchema')"
			>
				<div class="btn-group" role="group">
					<input
						id="networkSchemaHttp"
						v-model="values.schema"
						type="radio"
						class="btn-check"
						name="networkSchema"
						value="http"
						autocomplete="off"
					/>
					<label class="btn btn-outline-primary" for="networkSchemaHttp">HTTP</label>
					<input
						id="networkSchemaHttps"
						v-model="values.schema"
						type="radio"
						class="btn-check"
						name="networkSchema"
						value="https"
						autocomplete="off"
					/>
					<label class="btn btn-outline-primary" for="networkSchemaHttps">HTTPS</label>
				</div>
			</FormRow>

			<FormRow
				id="networkHost"
				:label="$t('config.network.labelHost')"
				:help="$t('config.network.descriptionHost')"
				example="evcc.local"
			>
				<input id="networkHost" v-model="values.host" class="form-control" />
			</FormRow>
			<FormRow
				id="networkPort"
				:label="$t('config.network.labelPort')"
				:help="$t('config.network.descriptionPort')"
			>
				<input
					id="networkPort"
					v-model="values.port"
					class="form-control w-50 me-2 w-50"
					type="number"
					example="7070"
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
};
</script>
