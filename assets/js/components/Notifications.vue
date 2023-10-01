<template>
	<div>
		<button
			v-show="iconVisible"
			href="#"
			data-bs-toggle="modal"
			data-bs-target="#notificationModal"
			class="btn btn-sm btn-link text-decoration-none link-light text-nowrap"
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
								{{ fmtTimeAgo(msg.time - new Date()) }}
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
									{{ message(msg) }}
								</span>
								<span v-if="msg.count > 1" class="badge rounded-pill bg-secondary">
									{{ msg.count }}
								</span>
							</p>
						</div>
					</div>
					<div class="modal-footer">
						<button
							type="button"
							data-bs-dismiss="modal"
							aria-label="Close"
							class="btn btn-outline-secondary"
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

<script>
import "@h2d2/shopicons/es/regular/exclamationtriangle";
import formatter from "../mixins/formatter";

export default {
	name: "Notifications",
	mixins: [formatter],
	props: {
		notifications: Array,
		loadpointTitles: Array,
	},
	computed: {
		iconVisible: function () {
			return this.notifications.length > 0;
		},
		iconClass: function () {
			return this.notifications.find((m) => m.level === "error")
				? "text-danger"
				: "text-warning";
		},
	},
	created: function () {
		this.interval = setInterval(() => {
			this.$forceUpdate();
		}, 10 * 1000);
	},
	unmounted: function () {
		clearTimeout(this.interval);
	},
	methods: {
		message({ message, lp }) {
			let context = "";
			if (lp) {
				context = `${this.loadpointTitles[lp - 1] || lp}: `;
			}
			return `${context}${message}`;
		},
		clear: function () {
			window.app && window.app.clear();
		},
	},
};
</script>
