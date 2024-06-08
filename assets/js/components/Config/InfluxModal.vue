<template>
	<JsonModal
		id="influxModal"
		:title="$t('config.influx.title')"
		:description="$t('config.influx.description')"
		docs="/docs/reference/configuration/influx"
		endpoint="/config/influx"
		state-key="influx"
		data-testid="influx-modal"
		@changed="$emit('changed')"
	>
		<template v-slot:default="{ values }">
			<FormRow
				id="influxUrl"
				:label="$t('config.influx.labelUrl')"
				example="http://localhost:8086"
			>
				<input
					id="influxUrl"
					v-model="values.url"
					type="url"
					class="form-control"
					required
				/>
			</FormRow>
			<FormRow
				v-if="!showV1(values)"
				id="influxOrg"
				:label="$t('config.influx.labelOrg')"
				example="home"
			>
				<input id="influxOrg" v-model="values.org" class="form-control" required />
			</FormRow>
			<FormRow
				id="influxDatabase"
				:label="$t(`config.influx.label${showV1(values) ? 'Database' : 'Bucket'}`)"
				example="evcc"
			>
				<input
					id="influxDatabase"
					v-model="values.database"
					class="form-control"
					required
				/>
			</FormRow>
			<FormRow
				v-if="!showV1(values)"
				id="influxToken"
				:label="$t('config.influx.labelToken')"
				:help="$t('config.influx.descriptionToken')"
			>
				<input id="influxToken" v-model="values.token" class="form-control" required />
			</FormRow>
			<FormRow
				v-if="showV1(values)"
				id="influxUser"
				:label="$t('config.influx.labelUser')"
				optional
			>
				<input id="influxUser" v-model="values.user" class="form-control" />
			</FormRow>
			<FormRow
				v-if="showV1(values)"
				id="influxPassword"
				:label="$t('config.influx.labelPassword')"
				optional
			>
				<input
					id="influxPassword"
					v-model="values.password"
					class="form-control"
					type="password"
					autocomplete="off"
				/>
			</FormRow>
			<p>
				<button
					v-if="showV1(values)"
					class="btn btn-link btn-sm text-primary px-0"
					type="button"
					@click="
						values.user = '';
						values.password = '';
						v1 = false;
					"
				>
					{{ $t("config.influx.v2Support") }}
				</button>
				<button
					v-else
					class="btn btn-link btn-sm text-primary px-0"
					type="button"
					@click="
						values.token = '';
						v1 = true;
					"
				>
					{{ $t("config.influx.v1Support") }}
				</button>
			</p>
		</template>
	</JsonModal>
</template>

<script>
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";

export default {
	name: "InfluxModal",
	components: { FormRow, JsonModal },
	emits: ["changed"],
	data() {
		return { v1: false };
	},
	methods: {
		showV1(values) {
			return this.v1 || values.user || values.password;
		},
	},
};
</script>
