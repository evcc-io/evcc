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
			navigator.clipboard
				.writeText(this.content)
				.then(() => {
					this.copied = true;
					setTimeout(() => {
						this.copied = false;
					}, 2000); // Reset after 2 seconds
				})
				.catch((err) => {
					console.error("Failed to copy to clipboard:", err);
				});
		},
	},
});
</script>
