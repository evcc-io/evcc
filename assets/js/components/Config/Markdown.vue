<template>
	<!-- oxlint-disable-next-line vue/no-v-html -->
	<div class="root" v-html="compiledMarkdown"></div>
</template>

<script>
import { renderMarkdown } from "../../utils/markdown";

export default {
	name: "MarkdownRenderer",
	props: {
		markdown: String,
	},
	computed: {
		compiledMarkdown() {
			return renderMarkdown(this.markdown);
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
.root :deep(table) {
	display: block;
	overflow-x: auto;
	margin: 1em 0;
	border-collapse: collapse;
	hyphens: none;
}
.root :deep(th),
.root :deep(td) {
	padding: 0.25em 0.75em 0.25em 0;
	text-align: left;
	vertical-align: top;
	border-bottom: 1px solid var(--evcc-gray-50);
}
.root :deep(th) {
	color: var(--evcc-gray);
	font-weight: normal;
	white-space: nowrap;
}
</style>
