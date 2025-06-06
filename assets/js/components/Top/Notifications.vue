<template>
	<div>
		<button
			v-show="iconVisible"
			href="#"
			data-bs-toggle="modal"
			data-bs-target="#notificationModal"
			class="btn btn-sm btn-link text-decoration-none link-light border-0 text-nowrap"
			data-testid="notification-icon"
		>
			<shopicon-regular-exclamationtriangle
				:class="iconClass"
			></shopicon-regular-exclamationtriangle>
		</button>

		<div
			id="notificationModal"
			class="modal fade text-dark"
			tabindex="-1"
			role="dialog"
			aria-hidden="true"
			data-bs-backdrop="true"
		>
			<div
				class="modal-dialog modal-lg modal-dialog-centered modal-dialog-scrollable"
				role="document"
			>
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">{{ $t("notifications.modalTitle") }}</h5>
						<button
							type="button"
							class="btn-close"
							data-bs-dismiss="modal"
							aria-label="Close"
						></button>
					</div>
					<div class="modal-body">
						<div v-for="(msg, index) in notifications" :key="index">
							<small
								class="d-flex justify-content-end mt-3"
								:title="fmtAbsoluteDate(msg.time)"
							>
								{{ fmtTimeAgo(msg.time.getTime() - new Date().getTime()) }}
							</small>
							<p class="d-flex align-items-baseline">
								<shopicon-regular-exclamationtriangle
									:class="{
										'text-danger': msg.level === 'error',
										'text-warning': msg.level === 'warn',
									}"
									class="flex-grow-0 flex-shrink-0 d-block"
								></shopicon-regular-exclamationtriangle>
								<span class="flex-grow-1 px-2 py-1 text-break">
									<span
										v-for="(line, idx) in message(msg)"
										:key="idx"
										class="d-block"
									>
										{{ line }}
									</span>
								</span>
								<span v-if="msg.count > 1" class="badge rounded-pill bg-secondary">
									{{ msg.count }}
								</span>
							</p>
						</div>
					</div>
					<div class="modal-footer d-flex justify-content-between gap-3">
						<router-link to="/log" class="btn btn-outline-secondary">
							{{ $t("notifications.logs") }}
						</router-link>
						<button
							type="button"
							data-bs-dismiss="modal"
							aria-label="Close"
							class="btn btn-secondary"
							@click="clear"
						>
							{{ $t("notifications.dismissAll") }}
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/exclamationtriangle";
import formatter from "@/mixins/formatter";
import { defineComponent, type PropType } from "vue";
import type { Notification, Timeout } from "@/types/evcc";

export default defineComponent({
	name: "Notifications",
	mixins: [formatter],
	props: {
		notifications: { type: Array as PropType<Notification[]>, default: () => [] },
		loadpointTitles: { type: Array as PropType<string[]>, default: () => [] },
	},
	data() {
		return {
			interval: null as Timeout,
		};
	},
	computed: {
		iconVisible() {
			return this.notifications.length > 0;
		},
		iconClass() {
			return this.notifications.find((m) => m.level === "error")
				? "text-danger"
				: "text-warning";
		},
	},
	created() {
		this.interval = setInterval(() => {
			this.$forceUpdate();
		}, 10 * 1000);
	},
	unmounted() {
		if (this.interval) {
			clearTimeout(this.interval);
		}
	},
	methods: {
		message({ message, lp }: { message: string | string[]; lp?: number }) {
			const lines = Array.isArray(message) ? message : [message];
			if (lp) {
				// add loadpoint title to first line
				lines[0] = `${this.loadpointTitles[lp - 1] || lp}: ${lines[0]}`;
			}
			return lines;
		},
		clear() {
			window.app?.clear();
		},
	},
});
</script>
