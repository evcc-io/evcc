<template>
	<div>
		<h6 class="mb-3">{{ $t("config.remote.clients") }}</h6>
		<div
			v-for="client in clients"
			:key="client.username"
			data-testid="remote-client"
			class="mb-2"
		>
			<div
				class="d-flex align-items-center justify-content-between py-2 ps-3 pe-2 border rounded"
				:class="{ 'opacity-50': isExpired(client) }"
			>
				<div class="flex-grow-1 fw-semibold">{{ client.username }}</div>
				<small v-if="expiryLabel(client)" class="text-muted ms-2 me-2 text-end">
					{{ expiryLabel(client) }}
				</small>
				<button
					type="button"
					class="btn btn-sm btn-outline-secondary border-0"
					:aria-label="$t('config.remote.removeClient')"
					@click="$emit('remove', client.username)"
				>
					<shopicon-regular-trash size="s" class="flex-shrink-0"></shopicon-regular-trash>
				</button>
			</div>
		</div>

		<div v-if="!clients.length" class="text-muted small mb-2">
			{{ $t("config.remote.noClients") }}
		</div>

		<div class="d-flex justify-content-end mt-4">
			<button type="button" class="btn btn-outline-secondary" @click="$emit('add')">
				{{ $t("config.remote.addClient") }}
			</button>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import "@h2d2/shopicons/es/regular/trash";
import formatter from "@/mixins/formatter";
import minuteTicker from "@/mixins/minuteTicker";
import type { RemoteClient } from "@/types/evcc";

export default defineComponent({
	name: "RemoteClientList",
	mixins: [formatter, minuteTicker],
	props: {
		clients: { type: Array as PropType<RemoteClient[]>, required: true },
	},
	emits: ["add", "remove"],
	methods: {
		expiresInMs(client: RemoteClient): number | null {
			// reference everyMinute so dependents re-evaluate on each tick
			void this.everyMinute;
			if (!client.expiresAt) return null;
			return new Date(client.expiresAt).getTime() - Date.now();
		},
		isExpired(client: RemoteClient): boolean {
			const ms = this.expiresInMs(client);
			return ms !== null && ms <= 0;
		},
		expiryLabel(client: RemoteClient): string {
			const ms = this.expiresInMs(client);
			if (ms === null) return "";
			if (ms <= 0) return this.$t("config.remote.expired");
			return this.$t("config.remote.expiresIn", { time: this.fmtTimeAgo(ms) });
		},
	},
});
</script>
