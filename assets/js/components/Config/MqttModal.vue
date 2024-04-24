<template>
	<JsonModal
		id="mqttModal"
		:title="$t('config.mqtt.title')"
		:description="$t('config.mqtt.description')"
		docs="/docs/reference/configuration/mqtt"
		endpoint="/config/mqtt"
		data-testid="mqtt-modal"
		@changed="$emit('changed')"
	>
		<template v-slot:default="{ values }">
			<FormRow
				id="mqttBroker"
				:label="$t('config.mqtt.brokerLabel')"
				example="localhost:1883"
			>
				<input id="mqttBroker" v-model="values.Broker" class="form-control" required />
			</FormRow>

			<h6>{{ $t("config.mqtt.publishing") }}</h6>
			<FormRow
				id="mqttTopic"
				:label="$t('config.mqtt.topicLabel')"
				:help="$t('config.mqtt.topicDescription')"
				optional
			>
				<input id="mqttTopic" v-model="values.Topic" class="form-control" />
			</FormRow>
			<FormRow
				id="mqttClientId"
				:label="$t('config.mqtt.clientIdLabel')"
				:help="$t('config.mqtt.clientIdDescription')"
				optional
			>
				<input id="mqttClientId" v-model="values.ClientID" class="form-control" />
			</FormRow>

			<h6>{{ $t("config.mqtt.authentication") }}</h6>
			<FormRow id="mqttUser" :label="$t('config.mqtt.userLabel')" optional>
				<input id="mqttUser" v-model="values.User" class="form-control" />
			</FormRow>
			<FormRow id="mqttPassword" :label="$t('config.mqtt.passwordLabel')" optional>
				<input
					id="mqttPassword"
					v-model="values.Password"
					class="form-control"
					type="password"
					autocomplete="off"
				/>
			</FormRow>
			<FormRow id="mqttInsecure" :label="$t('config.mqtt.insecureLabel')">
				<div class="d-flex">
					<input
						class="form-check-input"
						id="mqttInsecure"
						type="checkbox"
						v-model="values.Insecure"
					/>
					<label class="form-check-label ms-2" for="mqttInsecure">
						{{ $t("config.mqtt.insecureCheckLabel") }}
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
