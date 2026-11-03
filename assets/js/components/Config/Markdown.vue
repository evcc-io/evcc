<template>
	<!-- oxlint-disable-next-line vue/no-v-html -->
	<div class="root" v-html="compiledMarkdown"></div>
</template>

<script>
import { marked } from "marked";

export default {
	name: "MarkdownRenderer",
	props: {
		markdown: String,
	},
	computed: {
		compiledMarkdown() {
			const html = marked.parse(this.markdown ?? "", { async: false });
			// open all links in new window
			return html.replace(/<a href=/g, '<a target="_blank" rel="noopener noreferrer" href=');
		},
	},
};
</script>
<style scoped>
.root {
	max-width: 100%;
}
.root > :deep(:last-child) {
	margin-bottom: 0;
}
.root :deep(pre) {
	overflow-x: auto;
	margin: 1em 0;
	hyphens: none;
}
</style>
