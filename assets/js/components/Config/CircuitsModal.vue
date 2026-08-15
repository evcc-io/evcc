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
		<template #default="{ values }: { values: Record<string, Circuit> }">
			<div
				v-if="Object.keys(values).length === 0"
				class="onboarding"
			>
				<p class="evcc-gray">
					No circuits configured. Start with a main circuit that
					represents your grid connection.
				</p>
				<button
					type="button"
					class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 mx-auto"
					tabindex="0"
					@click="openCircuit()"
				>
					<shopicon-regular-plus
						size="s"
						class="flex-shrink-0"
					></shopicon-regular-plus>
					Add main circuit
				</button>
			</div>
			<CircuitsTree :circuitsTree="circuitsTree(values)" />
		</template>
	</JsonModal>
</template>

<script lang="ts">
import JsonModal from "./JsonModal.vue";
import type { Circuit } from "@/types/evcc";
import CircuitsTree from "./CircuitsTree.vue";
import deepClone from "@/utils/deepClone.ts";
import { openModal } from "@/configModal.ts";

export interface RecursiveCircuit extends Circuit {
	circuitChilds?: RecursiveCircuit[];
}

export default {
	name: "CircuitsModal",
	components: { JsonModal, CircuitsTree },
	emits: ["changed"],
	methods: {
		circuitsTree(
			circuits: Record<string, RecursiveCircuit>,
		): RecursiveCircuit | undefined {
			let nodes = deepClone(circuits);

			let root: RecursiveCircuit | undefined;
			Object.entries(nodes).forEach(([_, node]) => {
				const parent = node.parent ? nodes[node.parent] : undefined;
				if (parent) {
					parent.circuitChilds ??= [];
					parent.circuitChilds.push(node);
				} else {
					// found the root
					root = node;
				}
			});

			return root;
		},
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
