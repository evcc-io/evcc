<template>
	<div v-if="compact" class="mt-4">
		<a href="#" role="button" class="text-danger" @click.prevent="open">
			{{ $t("config.sponsor.tokenRequiredShort") }}
		</a>
	</div>
	<Transition v-else name="collapse">
		<div v-if="shown" class="collapsible-wrapper open">
			<div class="collapsible-content ring-space">
				<div class="panel rounded-4 p-4 mb-4" data-testid="sponsor-token-required">
					<div class="d-flex align-items-center gap-2 fw-bold mb-1">
						<shopicon-regular-heart class="text-primary"></shopicon-regular-heart>
						{{ $t("config.sponsor.inline.title") }}
					</div>
					<p v-if="applied" class="mb-0">{{ $t("config.sponsor.inline.applied") }}</p>
					<template v-else>
						<p>{{ description }}</p>
						<form class="d-flex gap-2" @submit.prevent="apply">
							<input
								ref="field"
								v-model.trim="token"
								type="text"
								class="form-control font-monospace"
								:class="{ 'is-invalid': error }"
								:placeholder="$t('config.sponsor.inline.placeholder')"
								:aria-label="$t('config.sponsor.labelToken')"
								spellcheck="false"
								required
								@paste="handlePaste"
								@input="error = false"
							/>
							<button
								type="submit"
								class="btn btn-primary text-nowrap"
								:disabled="!token || saving"
							>
								<span
									v-if="saving"
									class="spinner-border spinner-border-sm me-2"
									role="status"
									aria-hidden="true"
								></span>
								{{ $t("config.sponsor.inline.apply") }}
							</button>
						</form>
						<p v-if="error" class="text-danger small mt-2 mb-0">
							{{ $t("config.sponsor.error") }}
						</p>
						<p class="small text-muted mt-3 mb-0">
							{{ $t("config.sponsor.inline.trial") }}
							<a :href="trialTokenLink" target="_blank"
								>{{ $t("config.sponsor.inline.trialLink") }}.</a
							>
							<br />
							<i18n-t keypath="config.sponsor.inline.learn" tag="span" scope="global">
								<template #becomeSponsor>
									<a href="https://sponsor.evcc.io" target="_blank">
										{{ $t("config.sponsor.inline.becomeSponsor") }}
									</a>
								</template>
							</i18n-t>
						</p>
					</template>
				</div>
			</div>
		</div>
	</Transition>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/heart";
import { defineComponent } from "vue";
import confetti from "canvas-confetti";
import api, { allowClientError } from "@/api";
import { openModal } from "@/configModal";
import { docsPrefix } from "@/i18n";
import { cleanYaml } from "@/utils/cleanYaml";

// applies the token in place, confirms for a moment and collapses once the sponsor status arrived
export default defineComponent({
	name: "SponsorTokenRequired",
	props: {
		compact: Boolean,
		feature: Boolean,
		isSponsor: Boolean,
	},
	data() {
		return {
			token: "",
			saving: false,
			error: false,
			applied: false,
			dismissed: false,
		};
	},
	computed: {
		description(): string {
			return this.$t(`config.sponsor.inline.${this.feature ? "feature" : "device"}`);
		},
		trialTokenLink(): string {
			return `${docsPrefix()}/sponsorship#trial`;
		},
		// the sponsor status arrives over the websocket before the request resolves, so stay while saving
		shown(): boolean {
			return this.applied || this.saving ? !this.dismissed : !this.isSponsor;
		},
	},
	methods: {
		open() {
			openModal("sponsor");
		},
		handlePaste(event: ClipboardEvent) {
			event.preventDefault();
			const text = event.clipboardData?.getData("text") ?? "";
			this.token = cleanYaml(text, "sponsortoken").trim();
		},
		async apply() {
			this.saving = true;
			this.error = false;
			try {
				const res = await api.post(
					"/config/sponsortoken",
					{ token: this.token },
					allowClientError
				);
				this.error = res.status >= 400;
				if (this.error) return;
				this.celebrate();
				this.applied = true;
				setTimeout(() => (this.dismissed = true), 5000);
			} finally {
				this.saving = false;
			}
		},
		// small primary burst out of the token field
		celebrate() {
			const { top, left, width, height } = (
				this.$refs["field"] as HTMLElement
			).getBoundingClientRect();
			confetti({
				origin: {
					x: (left + width / 2) / window.innerWidth,
					y: (top + height / 2) / window.innerHeight,
				},
				particleCount: 40,
				spread: 70,
				scalar: 0.8,
				zIndex: 2000,
				colors: [
					getComputedStyle(document.documentElement).getPropertyValue("--bs-primary"),
				],
			});
		},
	},
});
</script>

<style scoped>
.panel {
	background-color: var(--evcc-gray-10);
}
/* collapse on leave, vue removes the node afterwards so neighbour margins collapse again */
.collapse-leave-active.open {
	transition: grid-template-rows var(--evcc-transition-fast) ease var(--evcc-transition-fast);
}
.collapse-leave-to.open {
	grid-template-rows: 0fr;
}
.collapse-leave-to.open > .collapsible-content {
	opacity: 0;
	visibility: hidden;
	transition:
		opacity var(--evcc-transition-fast) ease,
		visibility var(--evcc-transition-fast) ease;
}
</style>
