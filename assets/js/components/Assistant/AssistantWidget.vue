<template>
	<div class="assistant safe-area-inset">
		<button
			v-if="!open"
			type="button"
			class="fab btn btn-primary d-flex align-items-center justify-content-center"
			:aria-label="$t('assistant.open')"
			data-testid="assistant-open"
			@click="open = true"
		>
			<AssistantIcon />
		</button>

		<div v-else class="panel d-flex flex-column" data-testid="assistant-panel">
			<div class="head d-flex align-items-center gap-2 px-3 py-2">
				<AssistantIcon class="flex-shrink-0" />
				<strong class="flex-grow-1 text-truncate">{{ $t("assistant.title") }}</strong>
				<button
					v-if="messages.length"
					type="button"
					class="btn btn-sm btn-link text-muted px-1"
					@click="reset"
				>
					{{ $t("assistant.clear") }}
				</button>
				<button
					type="button"
					class="btn-close"
					:aria-label="$t('assistant.close')"
					@click="open = false"
				></button>
			</div>

			<div ref="log" class="log flex-grow-1 overflow-auto px-3 py-2">
				<p v-if="!messages.length" class="text-muted mb-0">
					{{ $t("assistant.placeholder") }}
				</p>
				<div v-for="(m, i) in messages" :key="i" class="message" :class="m.role">
					<Markdown v-if="m.role === 'assistant'" :markdown="m.content" />
					<span v-else>{{ m.content }}</span>
				</div>
				<div v-if="pending" class="text-muted d-flex align-items-center gap-2">
					<span class="spinner-border spinner-border-sm" role="status"></span>
					{{ $t("assistant.thinking") }}
				</div>
				<p v-if="error" class="text-danger mb-0">{{ error }}</p>
			</div>

			<form class="foot d-flex gap-2 px-3 py-2" @submit.prevent="submit">
				<input
					ref="input"
					v-model="question"
					class="form-control"
					:placeholder="$t('assistant.inputPlaceholder')"
					:disabled="pending"
					data-testid="assistant-input"
				/>
				<button
					type="submit"
					class="btn btn-primary"
					:disabled="pending || !question.trim()"
				>
					{{ $t("assistant.send") }}
				</button>
			</form>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import AssistantIcon from "../MaterialIcon/Assistant.vue";
import Markdown from "../Config/Markdown.vue";
import api from "@/api";
import type { AssistantMessage } from "@/types/evcc";

export default defineComponent({
	name: "AssistantWidget",
	components: { AssistantIcon, Markdown },
	props: {
		// situational context handed to the model, e.g. the current page and visible errors
		context: { type: String as PropType<string>, default: "" },
	},
	data() {
		return {
			open: false,
			pending: false,
			question: "",
			error: "",
			messages: [] as AssistantMessage[],
		};
	},
	methods: {
		reset() {
			this.messages = [];
			this.error = "";
		},
		scrollDown() {
			this.$nextTick(() => {
				const log = this.$refs["log"] as HTMLElement | undefined;
				if (log) log.scrollTop = log.scrollHeight;
			});
		},
		async submit() {
			const content = this.question.trim();
			if (!content || this.pending) return;

			this.question = "";
			this.error = "";
			this.messages.push({ role: "user", content });
			this.pending = true;
			this.scrollDown();

			try {
				const res = await api.post(
					"/assistant/chat",
					{ messages: this.messages, context: this.context },
					{ validateStatus: (code) => [200, 400, 412, 500, 502].includes(code) }
				);
				if (res.status === 200) {
					this.messages.push({ role: "assistant", content: res.data.content });
				} else {
					this.error = res.data?.error || res.statusText;
				}
			} catch (e: any) {
				this.error = e.message;
			}
			this.pending = false;
			this.scrollDown();
		},
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";
.assistant {
	position: fixed;
	right: 1rem;
	/* clear the mobile tab bar */
	bottom: calc(5rem + env(safe-area-inset-bottom));
	z-index: 1030;
}
@media (--md-and-up) {
	.assistant {
		bottom: calc(1rem + env(safe-area-inset-bottom));
	}
}
.fab {
	width: 3.5rem;
	height: 3.5rem;
	border-radius: 50%;
	box-shadow: 0 0.25rem 1rem var(--evcc-gray-25);
}
.panel {
	width: min(24rem, calc(100vw - 2rem));
	height: min(32rem, calc(100vh - 9rem));
	background-color: var(--evcc-box);
	border: 1px solid var(--evcc-box-border);
	border-radius: 1rem;
	box-shadow: 0 0.25rem 1.5rem var(--evcc-gray-25);
	overflow: hidden;
}
.head,
.foot {
	border-color: var(--evcc-box-border);
	flex-shrink: 0;
}
.head {
	border-bottom: 1px solid var(--evcc-box-border);
}
.foot {
	border-top: 1px solid var(--evcc-box-border);
}
.message {
	margin-bottom: 0.75rem;
	overflow-wrap: anywhere;
}
.message.user {
	font-weight: bold;
}
</style>
