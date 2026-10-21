<template>
	<!-- oxlint-disable-next-line vue/no-v-html -->
	<div class="root" v-html="compiledMarkdown"></div>
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
			// snarkdown takes every underscore as emphasis; keep the ones inside words
			// like charge_before_export as text (private use char, restored after rendering)
			const html = snarkdown(this.markdown.replace(/(?<=[^\W_])_(?=[^\W_])/g, "\uE000"));
			// open all links in new window
			return html
				.replace(/\uE000/g, "_")
				.replace(/<a href=/g, '<a target="_blank" rel="noopener noreferrer" href=');
		},
	},
};
</script>
<style scoped>
.root {
	max-width: 100%;
}
.root :deep(pre.code) {
	overflow-x: auto;
	margin: 1em 0;
	hyphens: none;
}
</style>
