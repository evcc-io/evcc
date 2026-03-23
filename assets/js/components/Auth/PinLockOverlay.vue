<template>
	<div
		class="pin-lock-overlay"
		role="dialog"
		aria-modal="true"
		:aria-label="$t('main.uilock.title')"
	>
		<div class="pin-lock-card card">
			<div class="pin-lock-body">
				<div class="pin-lock-side-info">
					<p class="pin-lock-title text-center fw-bold mb-0">
						{{ $t("main.uilock.title") }}
					</p>
					<input
						ref="pinInput"
						class="pin-display font-monospace text-center"
						type="password"
						inputmode="numeric"
						:value="pin"
						:placeholder="$t('main.uilock.pinInput')"
						autocomplete="one-time-code"
						autocorrect="off"
						spellcheck="false"
						aria-live="polite"
						@input="onDirectInput"
						@keydown.enter="submit"
					/>
					<p
						v-if="error"
						class="text-danger text-center mb-0 pin-lock-error"
					>
						{{ error }}
					</p>
					<button
						type="button"
						class="btn btn-primary w-100 pin-lock-submit"
						:disabled="pin.length < 1 || submitting"
						@click="submit"
					>
						<span
							v-if="submitting"
							class="spinner-border spinner-border-sm"
							role="status"
							aria-hidden="true"
						></span>
						{{ $t("main.uilock.unlock") }}
					</button>
				</div>
				<div class="pin-lock-side-pad">
					<div class="pin-lock-grid">
						<button
							v-for="n in 9"
							:key="n"
							type="button"
							class="btn btn-outline-secondary"
							@click="appendDigit(String(n))"
						>
							{{ n }}
						</button>
						<button
							type="button"
							class="btn btn-outline-secondary"
							@click="appendDigit('0')"
						>
							0
						</button>
						<button
							type="button"
							class="btn btn-outline-warning"
							@click="backspace"
						>
							⌫
						</button>
						<button
							type="button"
							class="btn btn-outline-danger"
							@click="clearPin"
						>
							C
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import api from "@/api";

export default {
	name: "PinLockOverlay",
	emits: ["unlocked"],
	data() {
		return {
			pin: "",
			submitting: false,
			error: "",
		};
	},
	methods: {
		appendDigit(d) {
			if (this.pin.length >= 12) return;
			this.error = "";
			this.pin += d;
		},
		backspace() {
			this.pin = this.pin.slice(0, -1);
			this.error = "";
		},
		clearPin() {
			this.pin = "";
			this.error = "";
		},
		onDirectInput(e) {
			const digits = (e.target.value || "").replace(/\D/g, "").slice(0, 12);
			this.pin = digits;
			e.target.value = this.pin;
			this.error = "";
		},
		async submit() {
			if (this.pin.length < 1 || this.submitting) return;
			this.submitting = true;
			this.error = "";
			try {
				const res = await api.post(
					"auth/uilock/unlock",
					{ pin: this.pin },
					{
						validateStatus: (s) => [204, 401].includes(s),
					}
				);
				if (res.status === 204) {
					this.$emit("unlocked");
					return;
				}
				this.pin = "";
				this.error = this.$t("main.uilock.wrongPin");
			} finally {
				this.submitting = false;
			}
		},
	},
};
</script>

<style scoped>
.pin-lock-overlay {
	position: fixed;
	inset: 0;
	z-index: 2000;
	box-sizing: border-box;
	width: 100%;
	height: 100dvh;
	overflow: hidden;
	display: grid;
	place-items: center;
	place-content: center;
	padding: max(0.5rem, env(safe-area-inset-top)) max(0.75rem, env(safe-area-inset-right))
		max(0.5rem, env(safe-area-inset-bottom)) max(0.75rem, env(safe-area-inset-left));
	background: rgba(0, 0, 0, 0.55);
	backdrop-filter: blur(2px);
}

.pin-lock-card {
	width: 80vw;
	height: 80vh;
	max-width: calc(100vw - 1rem);
	max-height: calc(100dvh - 1rem);
	overflow: hidden;
	padding: 1.5rem 2rem;
	font-size: 1rem;
	line-height: 1.35;
	border-radius: 1rem;
	border: none;
}

.pin-lock-body {
	display: grid;
	grid-template-columns: 1fr 2fr;
	gap: 2rem;
	align-items: center;
	height: 100%;
}

.pin-lock-side-info {
	display: flex;
	flex-direction: column;
	justify-content: center;
	gap: 0.5rem;
	height: 100%;
}

.pin-lock-title {
	font-size: 1.5em;
	line-height: 1.25;
}

.pin-display {
	font-size: 2rem;
	letter-spacing: 0.15em;
	min-height: 1.3em;
	line-height: 1.3;
	text-align: center;
	background: transparent;
	border: 1px solid var(--bs-border-color-translucent, rgba(0, 0, 0, 0.15));
	border-radius: 0.5rem;
	padding: 0.35rem 0.5rem;
	color: inherit;
	width: 100%;
	cursor: text;
}

/* hide browser-native password reveal toggle */
.pin-display::-ms-reveal,
.pin-display::-webkit-credentials-auto-fill-button {
	display: none;
}

.pin-display::placeholder {
	color: var(--evcc-gray, #93949e);
	opacity: 0.6;
	font-size: 0.6em;
	letter-spacing: 0;
}

.pin-display:focus {
	outline: none;
	border-color: var(--bs-primary, #0ba631);
	box-shadow: 0 0 0 0.2rem rgba(11, 166, 49, 0.2);
}

.pin-lock-side-pad {
	height: 100%;
	display: flex;
	align-items: stretch;
}

.pin-lock-grid {
	display: grid;
	grid-template-columns: repeat(3, 1fr);
	grid-template-rows: repeat(4, 1fr);
	gap: 0.5rem;
	width: 100%;
	height: 100%;
}

.pin-lock-grid .btn {
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 0;
	font-size: 1.5em;
	line-height: 1;
	border-radius: 0.5rem;
	min-height: 0;
}

.pin-lock-error {
	font-size: 0.9em;
	line-height: 1.3;
}

.pin-lock-submit {
	padding: 0.6rem;
	font-size: 1.1em;
	border-radius: 0.5rem;
}

/* Narrow / portrait: stack vertically */
@media (max-width: 480px) {
	.pin-lock-card {
		width: calc(100vw - 1rem);
		height: calc(100dvh - 1rem);
		padding: 1rem;
	}

	.pin-lock-body {
		grid-template-columns: 1fr;
		grid-template-rows: auto 1fr;
		gap: 0.75rem;
	}

	.pin-lock-side-info {
		height: auto;
	}

	.pin-lock-grid {
		font-size: 0.9rem;
	}
}
</style>
