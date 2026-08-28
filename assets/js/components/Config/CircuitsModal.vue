<template>
	<JsonModal
		id="circuitsModal"
		name="circuits"
		:title="$t('config.circuits.title')"
		:description="$t('config.circuits.description')"
		docs="/docs/features/loadmanagement"
		endpoint="/config/circuits"
		state-key="circuits"
		data-testid="circuits-modal"
		no-buttons
		@changed="$emit('changed')"
	>
		<template #default>
			<PlaceholderButton v-if="circuits.length === 0" @click="openCircuit()">
				<div>
					<p class="mb-3">{{ $t("config.circuits.noCircuitsConfigured") }}</p>
					<div class="d-flex align-items-center justify-content-center">
						<shopicon-regular-plus class="me-1"></shopicon-regular-plus>
						<span>{{ $t("config.circuits.addMainCircuit") }}</span>
					</div>
				</div>
			</PlaceholderButton>
			<div v-else>
				<CircuitsTree
					class="mb-3"
					:circuitsTree="configCircuitTree(circuits)"
					:on-add-sub="onAddSub"
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
import type { ConfigCircuit } from "@/types/evcc";
import CircuitsTree from "./CircuitsTree.vue";
import PlaceholderButton from "../Helper/PlaceholderButton.vue";
import "@h2d2/shopicons/es/regular/plus";
import { openModal } from "@/configModal.ts";
import { configCircuitTree } from "@/utils/circuits.ts";
import type { PropType } from "vue";

export default {
	name: "CircuitsModal",
	components: { JsonModal, CircuitsTree, PlaceholderButton },
	emits: ["changed"],
	props: {
		circuits: { type: Array as PropType<ConfigCircuit[]>, required: true },
		onAddSub: {
			type: Function as PropType<(parent?: string) => void>,
			required: true,
		},
	},
	methods: {
		configCircuitTree,
		async openCircuit() {
			await openModal("circuit");
		},
	},
};
</script>
