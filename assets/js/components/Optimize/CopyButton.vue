<template>
	<button
		class="btn btn-link position-absolute text-primary rounded"
		:style="buttonStyle"
		:title="copied ? 'Copied!' : 'Copy to clipboard'"
		@click="handleCopy"
	>
		<shopicon-regular-checkmark v-if="copied"></shopicon-regular-checkmark>
		<shopicon-regular-copy v-else></shopicon-regular-copy>
	</button>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import "@h2d2/shopicons/es/regular/copy";
import "@h2d2/shopicons/es/regular/checkmark";

export default defineComponent({
	name: "CopyButton",
	props: {
		content: {
			type: String,
			required: true,
		},
		top: {
			type: String,
			default: "1px",
		},
		right: {
			type: String,
			default: "1px",
		},
		padding: {
			type: String,
			default: "10px",
		},
	},
	data() {
		return {
			copied: false,
		};
	},
	computed: {
		buttonStyle() {
			return {
				top: this.top,
				right: this.right,
				padding: this.padding,
				backgroundColor: "var(--evcc-box)",
			};
		},
	},
	methods: {
		handleCopy() {
			// Modern browser API, requires secure context (localhost or https)
			if (navigator.clipboard && window.isSecureContext) {
				navigator.clipboard
					.writeText(this.content)
					.then(() => {
						this.showCopiedFeedback();
					})
					.catch((err) => {
						console.error("Failed to copy to clipboard:", err);
						this.fallbackCopy();
					});
			} else {
				// Fallback method for HTTP/non-secure contexts
				this.fallbackCopy();
			}
		},
		fallbackCopy() {
			try {
				// Create a temporary textarea element
				const textarea = document.createElement("textarea");
				textarea.value = this.content;
				textarea.style.position = "fixed";
				textarea.style.left = "-999999px";
				textarea.style.top = "-999999px";
				document.body.appendChild(textarea);

				// Select and copy the text
				textarea.focus();
				textarea.select();
				const successful = document.execCommand("copy");
				document.body.removeChild(textarea);

				if (successful) {
					this.showCopiedFeedback();
				} else {
					alert("Copy failed. Please copy manually.");
				}
			} catch (err) {
				console.error("Fallback copy failed:", err);
				// Show user feedback that copy failed
				alert("Copy failed. Please copy manually.");
			}
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
