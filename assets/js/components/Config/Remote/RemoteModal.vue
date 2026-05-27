<template>
	<GenericModal
		id="remoteModal"
		:title="modalTitle"
		config-modal-name="remote"
		data-testid="remote-modal"
		@open="onOpen"
	>
		<div class="alert alert-warning">
			Development preview. Not ready for general use. Use with caution and monitor your system
			closely. Feedback welcome!
		</div>
		<SponsorTokenRequired v-if="!isSponsor" feature class="mt-0" />
		<ErrorMessage :error="error" />

		<template v-if="view === 'list'">
			<p>{{ $t("config.remote.description") }}</p>
			<div class="form-check form-switch my-3">
				<input
					id="remoteEnabled"
					:checked="config.enabled"
					class="form-check-input"
					type="checkbox"
					role="switch"
					:disabled="!isSponsor"
					@change="change"
				/>
				<div class="form-check-label">
					<label for="remoteEnabled">{{ $t("config.remote.enableLabel") }}</label>
					<div v-if="status.url">
						<span v-if="status.connected" class="text-primary">
							{{ $t("config.remote.connected") }}
						</span>
						<span v-else class="text-muted small">
							{{ $t("config.remote.disconnected") }}
						</span>
					</div>
				</div>
			</div>

			<div v-if="status.url" class="mt-4">
				<FormRow id="remoteServerUrl" :label="$t('config.remote.url')">
					<input
						id="remoteServerUrl"
						type="text"
						class="form-control border"
						:value="status.url"
						readonly
					/>
				</FormRow>

				<hr class="my-4" />

				<div v-if="status.loginBlocked" class="alert alert-danger">
					{{ $t("config.remote.loginBlocked") }}
				</div>

				<RemoteClientList
					:clients="clients"
					:last-seen="status.lastSeen"
					:connected="status.connected"
					@add="view = 'create'"
					@remove="removeClient"
				/>
			</div>
		</template>

		<RemoteClientCreate
			v-else-if="view === 'create'"
			@cancel="view = 'list'"
			@submit="submitCreate"
		/>

		<RemoteClientReveal
			v-else-if="view === 'reveal' && createdClient && status.url"
			:client="createdClient"
			:server-url="status.url"
			@done="finishReveal"
		/>
	</GenericModal>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import GenericModal from "../../Helper/GenericModal.vue";
import ErrorMessage from "../../Helper/ErrorMessage.vue";
import FormRow from "../FormRow.vue";
import SponsorTokenRequired from "../DeviceModal/SponsorTokenRequired.vue";
import RemoteClientList from "./RemoteClientList.vue";
import RemoteClientCreate from "./RemoteClientCreate.vue";
import RemoteClientReveal from "./RemoteClientReveal.vue";
import api from "@/api";
import type {
	Remote,
	RemoteConfig,
	RemoteStatus,
	RemoteClient,
	RemoteClientCreated,
} from "@/types/evcc";
import type { AxiosError } from "axios";

type View = "list" | "create" | "reveal";

export default defineComponent({
	name: "RemoteModal",
	components: {
		GenericModal,
		ErrorMessage,
		FormRow,
		SponsorTokenRequired,
		RemoteClientList,
		RemoteClientCreate,
		RemoteClientReveal,
	},
	props: {
		remote: { type: Object as () => Remote | undefined, default: undefined },
		isSponsor: Boolean,
	},
	data() {
		return {
			error: null as string | null,
			view: "list" as View,
			clients: [] as RemoteClient[],
			createdClient: null as RemoteClientCreated | null,
		};
	},
	computed: {
		config(): RemoteConfig {
			return this.remote?.config ?? { enabled: false };
		},
		status(): RemoteStatus {
			return this.remote?.status ?? { connected: false, loginBlocked: false };
		},
		modalTitle(): string {
			switch (this.view) {
				case "create":
					return this.$t("config.remote.addClientTitle");
				case "reveal":
					return this.$t("config.remote.clientCreated");
				default:
					return `${this.$t("config.remote.title")} 🧪`;
			}
		},
	},
	methods: {
		async onOpen() {
			this.error = null;
			this.view = "list";
			this.createdClient = null;
			await this.loadClients();
		},
		async loadClients() {
			if (!this.status.url) {
				this.clients = [];
				return;
			}
			try {
				const res = await api.get("config/remote/clients");
				this.clients = res.data || [];
			} catch (err) {
				this.handleError(err);
			}
		},
		async change(e: Event) {
			const target = e.target as HTMLInputElement;
			const checked = target.checked;
			try {
				this.error = null;
				await api.post(`config/remote/${checked}`);
				if (checked) {
					await this.loadClients();
				}
			} catch (err) {
				target.checked = !checked;
				this.handleError(err);
			}
		},
		async submitCreate(payload: { username: string; expiresIn: number }) {
			try {
				this.error = null;
				const res = await api.post("config/remote/clients", payload);
				this.createdClient = res.data as RemoteClientCreated;
				this.view = "reveal";
			} catch (err) {
				this.handleError(err);
			}
		},
		async finishReveal() {
			this.createdClient = null;
			await this.loadClients();
			this.view = "list";
		},
		async removeClient(username: string) {
			if (!window.confirm(this.$t("config.remote.confirmDelete"))) {
				return;
			}
			try {
				this.error = null;
				await api.delete("config/remote/clients", { params: { username } });
				await this.loadClients();
			} catch (err) {
				this.handleError(err);
			}
		},
		handleError(err: unknown) {
			const axiosErr = err as AxiosError<{ error: string }>;
			this.error = axiosErr.response?.data?.error || axiosErr.message || String(err);
		},
	},
});
</script>
