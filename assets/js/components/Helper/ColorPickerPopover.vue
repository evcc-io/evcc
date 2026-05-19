<template>
	<Teleport to="body">
		<div
			v-if="modelValue"
			ref="popoverEl"
			class="color-picker-popover shadow"
			role="dialog"
			:aria-label="title || 'Color picker'"
			@click.stop
		>
			<div class="footer-row">
				<button
					type="button"
					class="auto-btn"
					:class="{ 'is-selected': mode === 'auto' }"
					@click="pickAuto"
				>
					{{ $t("colors.auto") }}
				</button>
				<div class="custom-input-wrap" :class="{ 'is-selected': mode === 'custom' }">
					<label
						class="custom-preview"
						:style="{ backgroundColor: customHex || '#00000000' }"
					>
						<input
							type="color"
							class="custom-native"
							:value="nativeColorValue"
							@input="onNativeInput"
						/>
					</label>
					<span class="custom-hash">#</span>
					<input
						ref="hexInputEl"
						type="text"
						class="custom-input"
						:value="hexInputValue"
						:aria-label="$t('colors.hex')"
						spellcheck="false"
						maxlength="8"
						@input="onHexInput"
						@focus="hexFocused = true"
						@blur="onHexBlur"
						@keydown.enter.prevent="onHexCommit"
					/>
				</div>
			</div>
			<div class="swatch-grid">
				<button
					v-for="hex in palette"
					:key="hex"
					type="button"
					class="swatch"
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
import colors from "../../colors";

const HEX_RE = /^[0-9a-fA-F]{6}([0-9a-fA-F]{2})?$/;

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
			palette: colors.palette,
			popper: null as PopperInstance | null,
			hexInputValue: "",
			hexFocused: false,
		};
	},
	computed: {
		customHex(): string {
			return this.normalize(this.color || "");
		},
		nativeColorValue(): string {
			return this.customHex.slice(0, 7) || "#000000";
		},
		matchesPalette(): boolean {
			if (!this.customHex) return false;
			return this.palette.some(
				(p) => this.normalize(p).slice(0, 7) === this.customHex.slice(0, 7)
			);
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
			this.hexInputValue = this.normalize(c).replace(/^#/, "");
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
			return this.normalize(hex).slice(0, 7) === this.customHex.slice(0, 7);
		},
		normalize(s: string): string {
			if (!s) return "";
			let c = s.trim().toUpperCase();
			if (!c.startsWith("#")) c = "#" + c;
			return c;
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
	border: 1px solid var(--bs-border-color-translucent);
	border-radius: 0.75rem;
	padding: 0.6rem;
	width: 16rem;
}
.swatch-grid {
	display: grid;
	grid-template-columns: repeat(6, 1fr);
	gap: 0.5rem;
	margin-top: 0.6rem;
	padding-top: 0.55rem;
	border-top: 1px solid var(--bs-border-color-translucent);
}
.swatch {
	border: none;
	padding: 0;
	width: 100%;
	aspect-ratio: 1;
	border-radius: 0.4rem;
	cursor: pointer;
	position: relative;
	display: flex;
	align-items: center;
	justify-content: center;
	color: #fff;
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
	grid-template-columns: repeat(6, 1fr);
	gap: 0.35rem;
	align-items: stretch;
}
.auto-btn {
	grid-column: span 2;
}
.custom-input-wrap {
	grid-column: span 4;
}
.auto-btn {
	border: 1px solid var(--bs-border-color-translucent);
	background: var(--bs-body-bg);
	color: inherit;
	border-radius: 0.4rem;
	padding: 0.2rem 0.55rem;
	cursor: pointer;
	font-size: 0.7rem;
	text-transform: uppercase;
	letter-spacing: 0.06em;
	font-weight: 600;
	min-width: 0;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}
.auto-btn.is-selected {
	border-color: var(--evcc-default-text);
}
.custom-input-wrap {
	display: flex;
	align-items: center;
	background: var(--bs-body-bg);
	border: 1px solid var(--bs-border-color-translucent);
	border-radius: 0.4rem;
	padding: 0.15rem 0.5rem;
	gap: 0.4rem;
	min-width: 0;
}
.custom-input-wrap.is-selected {
	border-color: var(--evcc-default-text);
}
.custom-preview {
	width: 1.2rem;
	height: 1.2rem;
	border-radius: 0.3rem;
	border: 1px solid var(--bs-border-color-translucent);
	position: relative;
	cursor: pointer;
	overflow: hidden;
	flex-shrink: 0;
}
.custom-native {
	position: absolute;
	inset: 0;
	opacity: 0;
	cursor: pointer;
	border: none;
	padding: 0;
	background: transparent;
}
.custom-hash {
	color: var(--bs-gray-medium);
}
.custom-input {
	flex: 1;
	width: 7ch;
	border: none;
	background: transparent;
	color: inherit;
	font-family: var(--bs-font-monospace);
	font-size: 0.8rem;
	outline: none;
	padding: 0.15rem 0;
	min-width: 0;
	text-transform: uppercase;
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
