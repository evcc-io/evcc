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
		disable-remove
		@changed="$emit('changed')"
	>
		<template #default>
			<div v-if="circuits.length === 0" class="onboarding">
				<p class="evcc-gray">
					{{ $t("config.circuits.noCircuitsConfigured") }}
				</p>
				<button
					type="button"
					class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 mx-auto"
					tabindex="0"
					@click="openCircuit()"
				>
					<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
					{{ $t("config.circuits.addMainCircuit") }}
				</button>
			</div>
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
		</template>
	</JsonModal>
</template>

<script lang="ts">
import JsonModal from "./JsonModal.vue";
import type { ConfigCircuit } from "@/types/evcc";
import CircuitsTree from "./CircuitsTree.vue";
import { openModal } from "@/configModal.ts";
import { configCircuitTree } from "@/utils/circuits.ts";
import type { PropType } from "vue";

export default {
	name: "CircuitsModal",
	components: { JsonModal, CircuitsTree },
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
<style scoped>
.onboarding {
	border: 1px dashed var(--evcc-gray-25);
	border-radius: 12px;
	padding: 20px;
	text-align: center;
}
</style>
