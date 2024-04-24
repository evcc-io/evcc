<template>
	<JsonModal
		id="influxModal"
		:title="$t('config.influx.title')"
		:description="$t('config.influx.description')"
		docs="/docs/reference/configuration/influx"
		endpoint="/config/influx"
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
					v-model="values.URL"
					type="url"
					class="form-control"
					required
				/>
			</FormRow>
			<FormRow id="influxDatabase" :label="$t('config.influx.labelDatabase')" example="evcc">
				<input
					id="influxDatabase"
					v-model="values.Database"
					class="form-control"
					required
				/>
			</FormRow>
			<div v-if="!showV1(values)">
				<FormRow
					id="influxToken"
					:label="$t('config.influx.labelToken')"
					:help="$t('config.influx.descriptionToken')"
				>
					<input id="influxToken" v-model="values.Token" class="form-control" required />
				</FormRow>
				<p>
					<button
						class="btn btn-link btn-sm text-primary px-0"
						type="button"
						@click="
							values.Token = '';
							v1 = true;
						"
					>
						{{ $t("config.influx.v1Support") }}
					</button>
				</p>
			</div>
			<div v-else>
				<FormRow id="influxUser" :label="$t('config.influx.labelUser')" optional>
					<input id="influxUser" v-model="values.User" class="form-control" />
				</FormRow>
				<FormRow id="influxPassword" :label="$t('config.influx.labelPassword')" optional>
					<input
						id="influxPassword"
						v-model="values.Password"
						class="form-control"
						type="password"
						autocomplete="off"
					/>
				</FormRow>
				<p>
					<button
						class="btn btn-link btn-sm text-primary px-0"
						type="button"
						@click="
							values.User = '';
							values.Password = '';
							v1 = false;
						"
					>
						{{ $t("config.influx.v2Support") }}
					</button>
				</p>
			</div>
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
			return this.v1 || values.User || values.Password;
		},
	},
};
</script>
