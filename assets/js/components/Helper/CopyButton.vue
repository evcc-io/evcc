<template>
	<slot :copy="handleCopy" :copied="copied" :copying="copying"></slot>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "CopyButton",
	props: {
		content: {
			type: String,
			required: true,
		},
		targetElement: {
			type: Object,
			default: null,
		},
	},
	data() {
		return {
			copied: false,
			copying: false,
		};
	},
	methods: {
		async handleCopy() {
			if (this.copying) return;

			this.copying = true;

			try {
				// Modern browser API - works in secure contexts and PWAs
				if (this.hasClipboardAPI()) {
					await navigator.clipboard.writeText(this.content);
					this.showCopiedFeedback();
				} else {
					// Fallback method for HTTP/non-secure contexts
					this.fallbackCopy();
				}
			} catch (err) {
				console.error("Failed to copy to clipboard:", err);
				this.fallbackCopy();
			} finally {
				this.copying = false;
			}
		},

		fallbackCopy() {
			try {
				let targetEl = this.targetElement as HTMLTextAreaElement;
				let createdElement = false;

				if (!targetEl) {
					// Create a temporary textarea element if no target provided
					targetEl = document.createElement("textarea");
					targetEl["value"] = this.content;
					targetEl["style"]["position"] = "fixed";
					targetEl["style"]["left"] = "-999999px";
					targetEl["style"]["top"] = "-999999px";
					document.body.appendChild(targetEl as Node);
					createdElement = true;
				}

				// Select and copy the text
				targetEl["focus"]();
				targetEl["select"]();

				// For input/textarea elements, also try setSelectionRange
				if (targetEl["setSelectionRange"]) {
					targetEl["setSelectionRange"](0, targetEl["value"].length);
				} else if ((targetEl as any)["createTextRange"]) {
					// IE fallback
					const range = (targetEl as any)["createTextRange"]();
					range.select();
				}

				const successful = document.execCommand("copy");

				if (createdElement) {
					document.body.removeChild(targetEl as Node);
				}

				if (successful) {
					this.showCopiedFeedback();
				} else {
					alert("Copy failed. Please copy manually.");
				}
			} catch (err) {
				console.error("Fallback copy failed:", err);
				alert("Copy failed. Please copy manually.");
			}
		},

		isSecureContext() {
			return (
				window.isSecureContext ||
				window.location.protocol === "https:" ||
				window.location.hostname === "localhost"
			);
		},

		isPWA() {
			// Check if running as PWA (installed web app)
			return (
				window.matchMedia("(display-mode: standalone)").matches ||
				(window.navigator as any).standalone === true || // iOS Safari
				document.referrer.includes("android-app://")
			); // Android
		},

		hasClipboardAPI() {
			// PWAs get clipboard access even over HTTP in many cases
			return navigator.clipboard && (this.isSecureContext() || this.isPWA());
		},

		showCopiedFeedback() {
			this.copied = true;
			setTimeout(() => {
				this.copied = false;
			}, 2000);
		},
	},
});
</script>
