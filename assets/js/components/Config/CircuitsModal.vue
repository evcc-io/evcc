<template>
	<JsonModal
		id="circuitsModal"
		name="circuits"
		:title="$t('config.circuits.title')"
		:description="$t('config.circuits.description')"
		docs="/features/loadmanagement"
		endpoint="/config/circuits"
		state-key="circuits"
		data-testid="circuits-modal"
		no-buttons
		@changed="$emit('changed')"
	>
		<template #default>
			<NewDeviceButton
				v-if="circuits.length === 0"
				:title="$t('config.circuits.addMainCircuit')"
				@click="openCircuit()"
			/>
			<div v-else>
				<CircuitsTree
					class="mb-3"
					:circuitsTree="configCircuitTree(circuits)"
					:meters="meters"
					:grid-meter="gridMeter"
				/>
				<span class="evcc-gray">
					{{ $t("config.circuits.chargingPointsNote") }}
				</span>
			</div>
			<div class="d-flex justify-content-end mt-4">
				<button type="button" class="btn btn-outline-primary px-4" data-bs-dismiss="modal">
					{{ $t("config.general.close") }}
				</button>
			</div>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import JsonModal from "./JsonModal.vue";
import type { ConfigCircuit, ConfigMeter } from "@/types/evcc";
import CircuitsTree from "./CircuitsTree.vue";
import NewDeviceButton from "./NewDeviceButton.vue";
import { openModal } from "@/configModal.ts";
import { configCircuitTree } from "@/utils/circuits.ts";
import type { PropType } from "vue";

export default {
	name: "CircuitsModal",
	components: { JsonModal, CircuitsTree, NewDeviceButton },
	emits: ["changed"],
	props: {
		circuits: { type: Array as PropType<ConfigCircuit[]>, required: true },
		meters: {
			type: Array as PropType<ConfigMeter[]>,
			default: () => [],
		},
		gridMeter: { type: Object as PropType<ConfigMeter> },
	},
	methods: {
		configCircuitTree,
		async openCircuit() {
			await openModal("circuit");
		},
	},
};
</script>
