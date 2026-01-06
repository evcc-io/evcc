<template>
	<!-- eslint-disable-next-line vue/no-v-html -->
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
			return snarkdown(this.markdown);
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
