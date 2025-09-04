<template>
	<span class="feed-in-pattern-indicator" :title="title"></span>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { feedInDisabledPattern } from "@/colors";

export default defineComponent({
	name: "FeedInPatternIndicator",
	props: {
		title: {
			type: String,
			default: "",
		},
	},
	computed: {
		patternCss() {
			return feedInDisabledPattern("var(--evcc-export)");
		},
	},
});
</script>

<style scoped>
.feed-in-pattern-indicator {
	display: inline-block;
	width: 1rem;
	height: 1rem;
	vertical-align: text-bottom;
	position: relative;
	border: 1px solid var(--evcc-export);
	border-radius: 2px;
	overflow: hidden;
}

.feed-in-pattern-indicator::before {
	content: "";
	position: absolute;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	background-image: v-bind(patternCss);
	opacity: 0.6;
}
</style>
