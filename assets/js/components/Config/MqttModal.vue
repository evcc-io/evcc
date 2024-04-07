<template>
	<GenericModal
		id="mqttModal"
		ref="modal"
		:title="$t('config.mqtt.title')"
		data-testid="mqtt-modal"
		@open="open"
	>
		<p class="text-danger" v-if="error">{{ error }}</p>
		<form ref="form" class="container mx-0 px-0">
			<FormRow
				id="mqttBroker"
				:label="$t('config.mqtt.brokerLabel')"
				example="localhost:1883"
			>
				<input id="mqttBroker" v-model="values.broker" class="form-control" />
			</FormRow>
			<FormRow id="mqttUser" :label="$t('config.mqtt.userLabel')" optional>
				<input id="mqttUser" v-model="values.user" class="form-control" />
			</FormRow>
			<FormRow id="mqttPassword" :label="$t('config.mqtt.passwordLabel')" optional>
				<input id="mqttPassword" v-model="values.password" class="form-control" />
			</FormRow>
			<FormRow
				id="mqttTopic"
				:label="$t('config.mqtt.topicLabel')"
				:help="$t('config.mqtt.topicDescription')"
				example="evcc"
				optional
			>
				<input id="mqttTopic" v-model="values.topic" class="form-control" />
			</FormRow>
			<div class="my-4 d-flex justify-content-between">
				<button type="button" class="btn btn-link text-muted" data-bs-dismiss="modal">
					{{ $t("config.general.cancel") }}
				</button>
				<button
					type="submit"
					class="btn btn-primary"
					:disabled="saving"
					@click.prevent="save"
				>
					<span
						v-if="saving"
						class="spinner-border spinner-border-sm"
						role="status"
						aria-hidden="true"
					></span>
					{{ $t("config.general.save") }}
				</button>
			</div>
		</form>
	</GenericModal>
</template>

<script>
import GenericModal from "../GenericModal.vue";
import FormRow from "./FormRow.vue";
import api from "../../api";

export default {
	name: "MqttModal",
	components: { FormRow, GenericModal },
	emits: ["changed"],
	data() {
		return {
			saving: false,
			error: "",
			values: {},
		};
	},

	methods: {
		async open() {
			await this.load();
		},
		async load() {
			try {
				const { data } = await api.get("/config/mqtt");
				this.values = data.result;
			} catch (e) {
				console.error(e);
			}
		},
		async save() {
			this.saving = true;
			this.error = "";
			try {
				await api.put("/config/site", this.values);
				this.$emit("changed");
				this.$refs.modal.close();
			} catch (e) {
				console.error(e);
				this.error = e.response.data.error;
			}
			this.saving = false;
		},
	},
};
</script>
<style scoped>
.container {
	margin-left: calc(var(--bs-gutter-x) * -0.5);
	margin-right: calc(var(--bs-gutter-x) * -0.5);
	padding-right: 0;
}
</style>
