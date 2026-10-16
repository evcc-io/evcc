<template>
	<Card :title="sponsor ? 'Explanation' : 'Feedback'" edge-to-edge class="box-pull-out">
		<template v-if="sponsor" #actions>
			<a href="https://evcc.io/en/datenschutz/" target="_blank" class="small text-muted"
				>Privacy</a
			>
		</template>
		<template v-if="sponsor">
			<p v-if="error" class="text-danger">{{ error }}</p>
			<template v-if="recent">
				<MarkdownRenderer :markdown="explanation" class="explanation mb-3" />
				<div
					class="d-flex flex-wrap justify-content-between column-gap-3 row-gap-1 small text-muted"
				>
					<span>
						{{ status }}
						<button
							v-if="explaining || !planExplained"
							type="button"
							class="btn-reset text-decoration-underline text-primary"
							:class="{ sweep: explaining }"
							:disabled="explaining"
							:aria-busy="explaining"
							data-testid="optimize-explain"
							@click="$emit('explain')"
						>
							{{ explaining ? "Updating explanation…" : "Update explanation" }}
						</button>
					</span>
					<span class="text-nowrap">
						<template v-if="voteSent">Thanks for the feedback.</template>
						<template v-else-if="!vote">
							Helpful?
							<button
								type="button"
								class="btn-reset text-decoration-underline"
								:disabled="voting"
								data-testid="optimize-explain-vote-up"
								@click="sendFeedback('up')"
							>
								Yes
							</button>
							<span class="mx-1" aria-hidden="true">·</span>
							<button
								type="button"
								class="btn-reset text-decoration-underline"
								:disabled="voting"
								data-testid="optimize-explain-vote-down"
								@click="sendFeedback('down')"
							>
								No
							</button>
						</template>
					</span>
				</div>
				<small v-if="voteError" class="d-block text-danger mt-2">{{ voteError }}</small>
				<form
					v-if="vote === 'down' && !voteSent"
					class="d-flex gap-2 mt-3"
					@submit.prevent="sendFeedback('down')"
				>
					<input
						v-model="voteComment"
						type="text"
						class="form-control form-control-sm"
						placeholder="What was off? (optional)"
						maxlength="1000"
					/>
					<button
						type="submit"
						class="btn btn-sm btn-outline-primary"
						:disabled="voting || !voteComment"
					>
						Send
					</button>
				</form>
			</template>
			<template v-else>
				<div class="d-flex flex-wrap align-items-center column-gap-3 row-gap-2">
					<button
						type="button"
						class="btn btn-outline-primary flex-shrink-0"
						:disabled="explaining"
						data-testid="optimize-explain"
						@click="$emit('explain')"
					>
						<span
							v-if="explaining"
							class="spinner-border spinner-border-sm me-2"
							role="status"
						></span>
						Explain plan
					</button>
					<small class="text-muted"
						>Lets an AI explain the above plan in plain words.</small
					>
				</div>
			</template>
		</template>
		<p class="small text-muted mb-0" :class="{ 'border-top mt-4 pt-3': sponsor }">
			Unexpected results or implausible numbers?
			<a href="https://github.com/evcc-io/optimizer/issues" target="_blank">Open an issue</a>
			and attach the request and response below.
		</p>
	</Card>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Card from "../Helper/Card.vue";
import MarkdownRenderer from "../Config/Markdown.vue";
import api from "@/api";
import formatter from "@/mixins/formatter";
import minuteTicker from "@/mixins/minuteTicker";
import type { EvOptExplanation } from "@/types/evcc";

const MAX_AGE = 60 * 60 * 1000;

// Plan explanation for sponsors and the feedback link for everyone. The explanation is
// kept per plan, a new one is offered once the plan updates.
export default defineComponent({
	name: "PlanExplanation",
	components: { Card, MarkdownRenderer },
	mixins: [formatter, minuteTicker],
	props: {
		sponsor: Boolean,
		explained: Object as PropType<EvOptExplanation>,
		plan: { type: String, default: "" }, // updated time of the current plan
		explaining: Boolean,
		error: { type: String, default: "" },
	},
	emits: ["explain"],
	data() {
		return {
			vote: "",
			voteComment: "",
			voting: false,
			voteSent: false,
			voteError: "",
		};
	},
	computed: {
		explanation(): string {
			return this.explained?.explanation ?? "";
		},
		explainedAt(): number {
			return this.explained ? new Date(this.explained.created).getTime() : 0;
		},
		planExplained(): boolean {
			return !!this.explained && this.explained.plan === this.plan;
		},
		// the ticker lags up to a minute, a fresh explanation must not read as written in the future
		now(): number {
			return Math.max(this.everyMinute.getTime(), this.explainedAt);
		},
		// older explanations are hidden, the plan has moved on too far for them to help
		recent(): boolean {
			return !!this.explanation && this.now - this.explainedAt < MAX_AGE;
		},
		status(): string {
			const written = `AI generated ${this.fmtTimeAgo(this.explainedAt - this.now)}`;
			return this.planExplained
				? `${written}. New explanation possible once the plan updates.`
				: `${written} for an earlier plan.`;
		},
	},
	methods: {
		// No counts at once, a comment follows as a repeat of the vote
		async sendFeedback(vote: string) {
			this.voting = true;
			this.voteError = "";
			try {
				await api.post("optimizer/explain/feedback", {
					vote,
					comment: this.voteComment,
				});
				this.vote = vote;
				this.voteSent = vote === "up" || !!this.voteComment;
			} catch (e: any) {
				this.voteError = e.response?.data?.error || e.message;
			}
			this.voting = false;
		},
	},
});
</script>

<style scoped>
/* snarkdown renders the blank lines around a list as bare <br>, the list margin replaces them */
.explanation :deep(ul) {
	margin: 1rem 0;
}
.explanation :deep(ul + br) {
	display: none;
}

/* busy state of the update link: a light sweeps across the text, same box as the idle link */
.sweep {
	background: linear-gradient(
		90deg,
		var(--bs-primary) 40%,
		var(--bs-primary-dim) 50%,
		var(--bs-primary) 60%
	);
	background-size: 300% 100%;
	background-clip: text;
	color: transparent !important;
	text-decoration: none;
	animation: sweep 1.5s linear infinite;
}
.sweep:disabled {
	opacity: 1;
}
@keyframes sweep {
	from {
		background-position: 100% 0;
	}
	to {
		background-position: 0 0;
	}
}
@media (prefers-reduced-motion: reduce) {
	.sweep {
		animation: none;
	}
}
</style>
