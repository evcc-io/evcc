<template>
	<DeviceModalBase
		device-type="meter"
		name="changeMeter"
		:modal-title="modalTitle"
		:initial-values="initialValues"
		:show-main-content="false"
	>
		<template #pre-content="{ values }">
			<FormRow
				id="meters"
				:label="$t('config.circuit.meterChangeLabel')"
				:help="$t('config.circuit.meterChangeHelp')"
			>
				<PropertyField
					id="circuitParamMeterChange"
					v-model="values.meter"
					type="Choice"
					size="w-100"
					class="me-2"
					:choice="meterOptions"
					required
				/>
			</FormRow>
		</template>
		<template #post-content="{ values }">
			<DeviceModalActions
				:is-deletable="meterId !== ''"
				:is-saveable="meterId !== values.meter"
				:can-disable="false"
				@save="selectMeter(values.meter)"
				@remove="removeMeter"
			/>
		</template>
	</DeviceModalBase>
</template>

<script lang="ts">
import { ConfigType, type ConfigMeter } from "@/types/evcc.ts";
import { defineComponent, type PropType } from "vue";
import FormRow from "./FormRow.vue";
import PropertyField from "./PropertyField.vue";
import DeviceModalBase from "./DeviceModal/DeviceModalBase.vue";
import DeviceModalActions from "./DeviceModal/Actions.vue";
import type { DeviceValues } from "./DeviceModal";
import { closeModal } from "@/configModal";

export default defineComponent({
	name: "ChangeMeterModal",
	components: { DeviceModalBase, DeviceModalActions, FormRow, PropertyField },
	props: {
		meters: {
			type: Array as PropType<ConfigMeter[]>,
			default: () => [],
		},
		meterId: {
			type: String,
			default: "",
		},
	},
	computed: {
		modalTitle() {
			if (this.meterId === "") {
				return this.$t("config.circuit.meterLabelAdd");
			} else {
				return this.$t("config.circuit.meterLabelEdit");
			}
		},
		initialValues(): DeviceValues {
			return {
				type: ConfigType.Template,
				template: null,
				meter: this.meterId,
			};
		},
		meterOptions() {
			return this.availableMeters.map((m) => ({
				key: m.name,
				name: `${m.deviceTitle} (${m.name})`,
			}));
		},
		availableMeters(): ConfigMeter[] {
			return this.meters.filter((m) => !!m.deviceTitle);
		},
	},
	methods: {
		async selectMeter(name: string) {
			await closeModal({ action: "added", name });
		},
		async removeMeter() {
			await closeModal({ action: "removed" });
		},
	},
});
</script>
