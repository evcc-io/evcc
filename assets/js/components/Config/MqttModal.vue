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
		<template #default="{ values }">
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
						id="mqttInsecure"
						v-model="values.insecure"
						class="form-check-input"
						type="checkbox"
					/>
					<label class="form-check-label ms-2" for="mqttInsecure">
						{{ $t("config.mqtt.labelCheckInsecure") }}
					</label>
				</div>
			</FormRow>
			<PropertyCollapsible>
				<template #advanced>
					<FormRow id="mqttCaCert" :label="$t('config.mqtt.labelCaCert')" optional>
						<PropertyCertField id="mqttCaCert" v-model="values.caCert" />
					</FormRow>
					<FormRow
						id="mqttClientCert"
						:label="$t('config.mqtt.labelClientCert')"
						optional
					>
						<PropertyCertField id="mqttClientCert" v-model="values.clientCert" />
					</FormRow>
					<FormRow id="mqttClientKey" :label="$t('config.mqtt.labelClientKey')" optional>
						<PropertyCertField id="mqttClientKey" v-model="values.clientKey" />
					</FormRow>
				</template>
			</PropertyCollapsible>
		</template>
	</JsonModal>
</template>

<script>
import JsonModal from "./JsonModal.vue";
import FormRow from "./FormRow.vue";
import PropertyCollapsible from "./PropertyCollapsible.vue";
import PropertyCertField from "./PropertyCertField.vue";

export default {
	name: "MqttModal",
	components: { FormRow, JsonModal, PropertyCollapsible, PropertyCertField },
	emits: ["changed"],
};
</script>
