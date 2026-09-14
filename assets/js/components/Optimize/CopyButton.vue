<template>
	<button
		class="btn btn-link position-absolute text-primary"
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
import { copyWithFeedback } from "@/utils/clipboard";
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
			};
		},
	},
	methods: {
		handleCopy() {
			copyWithFeedback(this.content, (value) => (this.copied = value));
		},
	},
});
</script>
