<template>
	<Teleport to="body">
		<div
			v-if="modelValue"
			ref="popoverEl"
			class="color-picker-popover shadow rounded-3 border"
			role="dialog"
			:aria-label="title || 'Color picker'"
			@click.stop
		>
			<div class="footer-row">
				<button
					type="button"
					class="auto-btn text-truncate text-uppercase fw-semibold text-body border rounded bg-body"
					:class="{ 'is-selected': mode === 'auto' }"
					@click="pickAuto"
				>
					{{ $t("colors.auto") }}
				</button>
				<div
					class="custom-input-wrap d-flex align-items-center border rounded bg-body"
					:class="{ 'is-selected': mode === 'custom' }"
				>
					<label
						class="custom-preview position-relative overflow-hidden border rounded flex-shrink-0"
						:style="{ backgroundColor: customHex || '#00000000' }"
					>
						<input
							type="color"
							class="custom-native position-absolute opacity-0 p-0 border-0 bg-transparent w-100 h-100 top-0 start-0"
							:value="nativeColorValue"
							@input="onNativeInput"
						/>
					</label>
					<span class="custom-hash text-secondary">#</span>
					<input
						ref="hexInputEl"
						type="text"
						class="custom-input border-0 bg-transparent font-monospace text-uppercase text-body flex-grow-1"
						:value="hexInputValue"
						:aria-label="$t('colors.hex')"
						spellcheck="false"
						maxlength="6"
						@input="onHexInput"
						@focus="hexFocused = true"
						@blur="onHexBlur"
						@keydown.enter.prevent="onHexCommit"
					/>
				</div>
			</div>
			<div class="swatch-grid border-top mt-2 pt-2">
				<button
					v-for="hex in palette"
					:key="hex"
					type="button"
					class="swatch border-0 p-0 d-flex align-items-center justify-content-center position-relative rounded text-white"
					:class="{ 'swatch--selected': isSwatchSelected(hex) }"
					:style="{ backgroundColor: hex }"
					:title="hex"
					@click="pick(hex)"
				>
					<svg
						v-if="isSwatchSelected(hex)"
						class="swatch-check"
						viewBox="0 0 16 16"
						width="14"
						height="14"
						aria-hidden="true"
					>
						<path
							d="M3.5 8.5l3 3 6-6"
							fill="none"
							stroke="currentColor"
							stroke-width="2.2"
							stroke-linecap="round"
							stroke-linejoin="round"
						/>
					</svg>
				</button>
			</div>
			<div class="popover-arrow" data-popper-arrow></div>
		</div>
	</Teleport>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { createPopper, type Instance as PopperInstance } from "@popperjs/core";
import colors, { HEX_RE, normalizeHex } from "../../colors";

export default defineComponent({
	name: "ColorPickerPopover",
	props: {
		modelValue: { type: Boolean, default: false },
		anchorEl: { type: Object as PropType<HTMLElement | null>, default: null },
		color: { type: String as PropType<string | null>, default: null },
		title: { type: String, default: "" },
	},
	emits: ["update:modelValue", "update:color"],
	data() {
		return {
			popper: null as PopperInstance | null,
			hexInputValue: "",
			hexFocused: false,
		};
	},
	computed: {
		palette(): string[] {
			// de-interleave warm/cool palette into 3×7 grid (rows: light/mid/dark, cols: hue)
			const grid: string[][] = Array.from({ length: 3 }, () => Array(7).fill(""));
			colors.palette.forEach((hex, i) => {
				const row = Math.floor(i / 7);
				const step = i % 7;
				const col = step < 6 ? Math.floor(step / 2) + (step % 2) * 3 : 6;
				grid[row][col] = hex;
			});
			return grid.flat();
		},
		customHex(): string {
			return normalizeHex(this.color);
		},
		nativeColorValue(): string {
			return this.customHex || "#000000";
		},
		matchesPalette(): boolean {
			if (!this.customHex) return false;
			return this.palette.some((p) => normalizeHex(p) === this.customHex);
		},
		mode(): "auto" | "palette" | "custom" {
			if (this.hexFocused) return "custom";
			if (!this.customHex) return "auto";
			if (this.matchesPalette) return "palette";
			return "custom";
		},
	},
	watch: {
		modelValue(open: boolean) {
			if (open) {
				this.hexInputValue = this.customHex.replace(/^#/, "");
				this.hexFocused = false;
				this.$nextTick(this.initPopper);
				document.addEventListener("mousedown", this.onDocClick, true);
				document.addEventListener("keydown", this.onKey, true);
			} else {
				this.teardown();
			}
		},
		color(c: string) {
			this.hexInputValue = normalizeHex(c).replace(/^#/, "");
		},
		anchorEl() {
			if (this.modelValue) this.$nextTick(this.initPopper);
		},
	},
	beforeUnmount() {
		this.teardown();
	},
	methods: {
		isSwatchSelected(hex: string): boolean {
			if (!this.customHex) return false;
			return normalizeHex(hex) === this.customHex;
		},
		pick(hex: string) {
			this.$emit("update:color", hex);
		},
		pickAuto() {
			this.hexFocused = false;
			(this.$refs["hexInputEl"] as HTMLInputElement | undefined)?.blur();
			this.$emit("update:color", "");
		},
		onHexInput(e: Event) {
			const v = (e.target as HTMLInputElement).value;
			this.hexInputValue = v;
			const trimmed = v.trim();
			if (HEX_RE.test(trimmed)) {
				this.$emit("update:color", "#" + trimmed.toUpperCase());
			}
		},
		onHexCommit() {
			const v = this.hexInputValue.trim();
			if (v === "") {
				this.$emit("update:color", "");
				return;
			}
			if (HEX_RE.test(v)) {
				this.$emit("update:color", "#" + v.toUpperCase());
			} else {
				this.hexInputValue = this.customHex.replace(/^#/, "");
			}
		},
		onHexBlur() {
			this.hexFocused = false;
			this.onHexCommit();
		},
		onNativeInput(e: Event) {
			const v = (e.target as HTMLInputElement).value;
			this.$emit("update:color", v.toUpperCase());
		},
		initPopper() {
			if (!this.anchorEl || !this.$refs["popoverEl"]) return;
			this.popper?.destroy();
			this.popper = createPopper(this.anchorEl, this.$refs["popoverEl"] as HTMLElement, {
				placement: "top",
				modifiers: [
					{ name: "offset", options: { offset: [0, 10] } },
					{ name: "arrow", options: { padding: 8 } },
					{ name: "preventOverflow", options: { padding: 8 } },
				],
			});
		},
		teardown() {
			this.popper?.destroy();
			this.popper = null;
			document.removeEventListener("mousedown", this.onDocClick, true);
			document.removeEventListener("keydown", this.onKey, true);
		},
		onDocClick(e: MouseEvent) {
			const t = e.target as Node;
			const inside =
				(this.$refs["popoverEl"] as HTMLElement | undefined)?.contains(t) ||
				this.anchorEl?.contains(t);
			if (!inside) this.close();
		},
		onKey(e: KeyboardEvent) {
			if (e.key === "Escape") this.close();
		},
		close() {
			this.$emit("update:modelValue", false);
		},
	},
});
</script>

<style scoped>
.color-picker-popover {
	z-index: 1080;
	background: var(--evcc-box);
	padding: 0.6rem;
	width: 16rem;
}
.swatch-grid {
	display: grid;
	grid-template-columns: repeat(7, 1fr);
	gap: 0.4rem;
}
.swatch {
	cursor: pointer;
	aspect-ratio: 1;
}
.swatch--selected {
	outline: 1px solid var(--evcc-default-text);
	outline-offset: 2px;
}
.swatch-check {
	pointer-events: none;
}
.footer-row {
	display: grid;
	grid-template-columns: 1fr 1fr;
	gap: 0.35rem;
}
.auto-btn {
	padding: 0.2rem 0.55rem;
	font-size: 0.7rem;
	letter-spacing: 0.06em;
	cursor: pointer;
}
.auto-btn.is-selected {
	border-color: var(--evcc-default-text) !important;
}
.custom-input-wrap {
	padding: 0.15rem 0.5rem;
	gap: 0.4rem;
	min-width: 0;
}
.custom-input-wrap.is-selected {
	border-color: var(--evcc-default-text) !important;
}
.custom-preview {
	width: 1.2rem;
	height: 1.2rem;
	cursor: pointer;
}
.custom-native {
	cursor: pointer;
}
.custom-input {
	width: 7ch;
	min-width: 0;
	font-size: 0.8rem;
	outline: none;
	padding: 0.15rem 0;
	color: inherit;
}
.popover-arrow {
	position: absolute;
	width: 12px;
	height: 12px;
}
.color-picker-popover[data-popper-placement^="top"] .popover-arrow {
	bottom: -6px;
}
.color-picker-popover[data-popper-placement^="bottom"] .popover-arrow {
	top: -6px;
}
.popover-arrow::before {
	content: "";
	position: absolute;
	width: 12px;
	height: 12px;
	background: var(--evcc-box);
	border-right: 1px solid var(--bs-border-color-translucent);
	border-bottom: 1px solid var(--bs-border-color-translucent);
	transform: rotate(45deg);
}
.color-picker-popover[data-popper-placement^="bottom"] .popover-arrow::before {
	transform: rotate(225deg);
	top: 0;
}
</style>
