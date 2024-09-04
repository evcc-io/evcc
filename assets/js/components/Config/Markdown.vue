<template>
	<div v-html="compiledMarkdown"></div>
</template>

<script>
import snarkdown from "snarkdown";

export default {
	name: "MarkdownRenderer",
	props: {
		markdown: String,
	},
	computed: {
		compiledMarkdown() {
			return snarkdown(this.markdown);
		},
	},
};
</script>
