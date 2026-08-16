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
		<template #default="{ values }: { values: State['circuits'] }">
			<div v-if="Object.keys(values?.config || {}).length === 0" class="onboarding">
				<p class="evcc-gray">
					No circuits configured. Start with a main circuit that represents your grid
					connection.
				</p>
				<button
					type="button"
					class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 mx-auto"
					tabindex="0"
					@click="openCircuit()"
				>
					<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
					Add main circuit
				</button>
			</div>
			<CircuitsTree
				v-else
				:circuitsTree="circuitTree(values?.config)"
				:on-add-sub="onAddSub"
			/>
		</template>
	</JsonModal>
</template>

<script lang="ts">
import JsonModal from "./JsonModal.vue";
import type { State } from "@/types/evcc";
import CircuitsTree from "./CircuitsTree.vue";
import { openModal } from "@/configModal.ts";
import { circuitTree } from "@/utils/circuits.ts";
import type { PropType } from "vue";

export default {
	name: "CircuitsModal",
	components: { JsonModal, CircuitsTree },
	emits: ["changed"],
	props: {
		onAddSub: {
			type: Function as PropType<(parent?: string) => void>,
			required: true,
		},
	},
	methods: {
		circuitTree,
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
