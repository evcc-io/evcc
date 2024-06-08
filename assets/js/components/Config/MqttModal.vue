<template>
	<JsonModal
		id="mqttModal"
		:title="$t('config.mqtt.title')"
		:description="$t('config.mqtt.description')"
		docs="/docs/reference/configuration/mqtt"
		endpoint="/config/mqtt"
		state-key="mqtt"
		data-testid="mqtt-modal"
		@changed="$emit('changed')"
	>
		<template v-slot:default="{ values }">
			<FormRow
				id="mqttBroker"
				:label="$t('config.mqtt.labelBroker')"
				example="localhost:1883"
			>
				<input id="mqttBroker" v-model="values.broker" class="form-control" required />
			</FormRow>

			<h6>{{ $t("config.mqtt.publishing") }}</h6>
			<FormRow
				id="mqttTopic"
				:label="$t('config.mqtt.labelTopic')"
				:help="$t('config.mqtt.descriptionTopic')"
				example="evcc"
				optional
			>
				<input id="mqttTopic" v-model="values.topic" class="form-control" />
			</FormRow>
			<FormRow
				id="mqttClientId"
				:label="$t('config.mqtt.labelClientId')"
				:help="$t('config.mqtt.descriptionClientId')"
				optional
			>
				<input id="mqttClientId" v-model="values.clientID" class="form-control" />
			</FormRow>

			<h6>{{ $t("config.mqtt.authentication") }}</h6>
			<FormRow id="mqttUser" :label="$t('config.mqtt.labelUser')" optional>
				<input id="mqttUser" v-model="values.user" class="form-control" />
			</FormRow>
			<FormRow id="mqttPassword" :label="$t('config.mqtt.labelPassword')" optional>
				<input
					id="mqttPassword"
					v-model="values.password"
					class="form-control"
					type="password"
					autocomplete="off"
				/>
			</FormRow>
			<FormRow id="mqttInsecure" :label="$t('config.mqtt.labelInsecure')">
				<div class="d-flex">
					<input
						class="form-check-input"
						id="mqttInsecure"
						type="checkbox"
						v-model="values.insecure"
					/>
					<label class="form-check-label ms-2" for="mqttInsecure">
						{{ $t("config.mqtt.labelCheckInsecure") }}
					</label>
				</div>
			</FormRow>
		</template>
	</JsonModal>
</template>

<script>
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";

export default {
	name: "MqttModal",
	components: { FormRow, JsonModal },
	emits: ["changed"],
};
</script>
