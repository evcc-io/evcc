<template>
	<div class="assistant-widget">
		<button
			v-if="!open"
			type="button"
			class="fab btn btn-primary d-flex align-items-center justify-content-center"
			:aria-label="$t('assistant.open')"
			data-testid="assistant-open"
			@click="open = true"
		>
			<AssistantIcon />
			<!-- the conversation runs on while minimized, so show what happened meanwhile -->
			<span
				v-if="pending"
				class="indicator spinner-border spinner-border-sm"
				role="status"
			></span>
			<span v-else-if="unread" class="indicator dot" data-testid="assistant-unread"></span>
		</button>

		<div v-else class="panel d-flex flex-column" data-testid="assistant-panel">
			<div class="head d-flex align-items-center gap-2 px-3 py-2">
				<AssistantIcon class="flex-shrink-0" />
				<strong class="flex-grow-1 text-truncate">{{ $t("assistant.title") }}</strong>
				<div class="form-check form-check-reverse m-0">
					<input
						id="assistantThinking"
						v-model="showThinking"
						class="form-check-input"
						type="checkbox"
						data-testid="assistant-thinking-toggle"
					/>
					<label class="form-check-label text-muted" for="assistantThinking">
						{{ $t("assistant.showThinking") }}
					</label>
				</div>
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
					@click="close"
				></button>
			</div>

			<div ref="log" class="log flex-grow-1 overflow-auto px-3 py-2">
				<p v-if="!messages.length" class="text-muted mb-0">
					{{ $t("assistant.placeholder") }}
				</p>
				<div v-for="(m, i) in messages" :key="i" class="message" :class="`message-${m.role}`">
					<template v-if="m.role === 'assistant'">
						<AssistantSteps v-if="showThinking" :steps="m.steps || []" />
						<Markdown :markdown="m.content" />
					</template>
					<span v-else>{{ m.content }}</span>
				</div>
				<div v-if="pending">
					<AssistantSteps v-if="showThinking" :steps="liveSteps" />
					<div class="text-muted d-flex align-items-center gap-2">
						<span class="spinner-border spinner-border-sm" role="status"></span>
						{{ $t("assistant.thinking") }}
					</div>
				</div>
				<p v-if="error" class="text-danger mb-0">{{ error }}</p>
			</div>

			<form class="foot d-flex gap-2 px-3 py-2" @submit.prevent="submit">
				<input
					ref="input"
					v-model="question"
					class="form-control"
					:placeholder="$t('assistant.inputPlaceholder')"
					data-testid="assistant-input"
					@keydown.up.prevent="historyBack"
					@keydown.down.prevent="historyForward"
					@keydown.esc.prevent="abort"
				/>
				<button
					v-if="pending"
					type="button"
					class="btn btn-outline-primary text-nowrap"
					data-testid="assistant-stop"
					@click="abort"
				>
					{{ $t("assistant.stop") }}
				</button>
				<button v-else type="submit" class="btn btn-primary" :disabled="!question.trim()">
					{{ $t("assistant.send") }}
				</button>
			</form>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import AssistantIcon from "../MaterialIcon/Assistant.vue";
import AssistantSteps from "./AssistantSteps.vue";
import Markdown from "../Config/Markdown.vue";
import api from "@/api";
import { openLoginModal } from "@/components/Auth/auth";
import settings from "@/settings";
import type {
	AssistantEvent,
	AssistantMessage,
	AssistantResult,
	AssistantStep,
} from "@/types/evcc";

export default defineComponent({
	name: "AssistantWidget",
	components: { AssistantIcon, AssistantSteps, Markdown },
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
			// steps of the running question, shown until the answer replaces them
			liveSteps: [] as AssistantStep[],
			// asked questions, oldest first, browsed with cursor up/down
			history: [] as string[],
			// position in history, -1 while editing the unsent draft
			historyIndex: -1,
			draft: "",
			// answer received while minimized
			unread: false,
			controller: null as AbortController | null,
		};
	},
	watch: {
		open(open: boolean) {
			if (!open) return;
			this.unread = false;
			this.focusInput();
			this.scrollDown();
		},
	},
	computed: {
		// ui only, the steps always arrive and are only shown on request
		showThinking: {
			get(): boolean {
				return settings.assistantThinking;
			},
			set(value: boolean) {
				settings.assistantThinking = value;
			},
		},
	},
	methods: {
		// close hides the panel, the conversation keeps running in the background
		close() {
			this.open = false;
		},
		reset() {
			this.abort();
			this.messages = [];
			this.error = "";
			this.focusInput();
		},
		focusInput() {
			this.$nextTick(() => (this.$refs["input"] as HTMLInputElement | undefined)?.focus());
		},
		// abort cancels the running question, the answer is not awaited any longer
		abort() {
			this.controller?.abort();
		},
		// historyBack steps to the previous question, keeping the draft for the way back
		historyBack() {
			if (this.historyIndex < 0) {
				if (!this.history.length) return;
				this.draft = this.question;
				this.historyIndex = this.history.length;
			}
			if (this.historyIndex === 0) return;
			this.historyIndex--;
			this.recall(this.history[this.historyIndex]);
		},
		// historyForward steps back towards the draft, stopping there
		historyForward() {
			if (this.historyIndex < 0) return;
			this.historyIndex++;
			if (this.historyIndex >= this.history.length) {
				this.historyIndex = -1;
				this.recall(this.draft);
			} else {
				this.recall(this.history[this.historyIndex]);
			}
		},
		recall(content: string) {
			this.question = content;
			this.$nextTick(() => {
				const input = this.$refs["input"] as HTMLInputElement | undefined;
				input?.setSelectionRange(content.length, content.length);
			});
		},
		scrollDown() {
			this.$nextTick(() => {
				const log = this.$refs["log"] as HTMLElement | undefined;
				if (log) log.scrollTop = log.scrollHeight;
			});
		},
		// readEvents hands over each ndjson line as it arrives, a line may be split across chunks
		async readEvents(body: ReadableStream<Uint8Array>, onEvent: (ev: AssistantEvent) => void) {
			const reader = body.getReader();
			const decoder = new TextDecoder();
			let buffer = "";

			for (;;) {
				const { done, value } = await reader.read();
				if (done) return;

				buffer += decoder.decode(value, { stream: true });
				const lines = buffer.split("\n");
				// the last piece is only complete once its newline arrives
				buffer = lines.pop() || "";

				for (const line of lines) {
					if (line.trim()) onEvent(JSON.parse(line));
				}
			}
		},
		async submit() {
			const content = this.question.trim();
			if (!content || this.pending) return;

			if (this.history[this.history.length - 1] !== content) {
				this.history.push(content);
			}
			this.historyIndex = -1;
			this.draft = "";

			this.question = "";
			this.error = "";
			this.liveSteps = [];
			this.messages.push({ role: "user", content });
			this.pending = true;
			this.scrollDown();

			const controller = new AbortController();
			this.controller = controller;
			const conversation = this.messages;

			try {
				const res = await fetch(`${api.defaults.baseURL}assistant/chat`, {
					method: "POST",
					headers: { "Content-Type": "application/json" },
					// steps are display only, sending them back would bloat every request
					body: JSON.stringify({
						messages: this.messages.map(({ role, content }) => ({ role, content })),
						context: this.context,
					}),
					signal: controller.signal,
				});

				if (!res.ok || !res.body) {
					if (res.status === 401) openLoginModal();
					const data = await res.json().catch(() => null);
					this.error = data?.error || res.statusText;
				} else {
					let result: AssistantResult | undefined;

					await this.readEvents(res.body, (ev) => {
						if (ev.step) {
							this.liveSteps.push(ev.step);
							this.scrollDown();
						}
						if (ev.error) this.error = ev.error;
						if (ev.result) result = ev.result;
					});

					if (result) {
						this.messages.push({
							role: "assistant",
							content: result.content,
							steps: result.steps,
						});
						if (!this.open) this.unread = true;
					}
				}
			} catch (e: any) {
				if (controller.signal.aborted) {
					// hand the question back for editing, unless the conversation was cleared meanwhile
					if (this.messages === conversation) {
						this.messages.pop();
						if (!this.question) this.recall(content);
					}
				} else {
					this.error = e.message;
				}
			}
			this.controller = null;
			this.pending = false;
			this.scrollDown();
		},
	},
});
</script>

<style scoped>
.assistant-widget {
	position: fixed;
	right: 1rem;
	/* --bottom-space clears the tab bar, which shares our z-index */
	bottom: calc(var(--bottom-space) + var(--safe-area-inset-bottom));
	z-index: 1030;
}
.fab {
	position: relative;
	width: 3.5rem;
	height: 3.5rem;
	border-radius: 50%;
	box-shadow: 0 0.25rem 1rem var(--evcc-gray-25);
}
.indicator {
	position: absolute;
	top: 0.4rem;
	right: 0.4rem;
}
.dot {
	width: 0.625rem;
	height: 0.625rem;
	border-radius: 50%;
	background-color: currentColor;
}
.panel {
	width: min(24rem, calc(100vw - 2rem));
	height: min(32rem, calc(100vh - var(--bottom-space) - 4rem));
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
.message-user {
	font-weight: bold;
}
</style>
