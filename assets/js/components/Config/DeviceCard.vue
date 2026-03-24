<template>
	<div
		class="root"
		:class="{
			'round-box': !unconfigured,
			'round-box--error': error,
			'round-box--warning': warning,
			'root--unconfigured': unconfigured,
			'root--with-tags': $slots.tags,
		}"
	>
		<div class="d-flex align-items-center" :class="{ 'mb-2': $slots.tags }">
			<div class="icon me-2">
				<slot name="icon" />
			</div>
			<strong
				class="flex-grow-1 text-nowrap text-truncate"
				data-bs-toggle="tooltip"
				:title="name"
				>{{ title }}</strong
			>
			<DeviceCardEditIcon
				:editable="editable"
				:noEditButton="noEditButton"
				:badge="badge"
				@edit="$emit('edit')"
			/>
		</div>
		<div v-if="$slots.tags" ref="tagsContainer" :style="tagsStyle">
			<hr class="my-3 divide" />
			<div ref="tagsContent">
				<slot name="tags" />
			</div>
		</div>
	</div>
</template>

<script>
import DeviceCardEditIcon from "./DeviceCardEditIcon.vue";
import settings from "../../settings";

export default {
	name: "DeviceCard",
	components: { DeviceCardEditIcon },
	props: {
		name: String,
		id: String,
		title: String,
		editable: Boolean,
		error: Boolean,
		unconfigured: Boolean,
		warning: Boolean,
		noEditButton: Boolean,
		badge: Boolean,
	},
	emits: ["edit"],
	data() {
		return {
			tagsMinHeight: null,
			resizeObserver: null,
		};
	},
	computed: {
		tagsStyle() {
			return this.tagsMinHeight ? { minHeight: `${this.tagsMinHeight}px` } : undefined;
		},
	},
	mounted() {
		if (!this.id) return;
		const cached = settings.cardHeights[this.id];
		if (cached > 0) {
			this.tagsMinHeight = cached;
		}
		// Cache tag heights to reduce layout shift. Hold cached min-height
		// until async data fills the space, then save and release.
		this.$nextTick(() => {
			const el = this.$refs.tagsContainer;
			const content = this.$refs.tagsContent;
			if (!el || !content) return;
			const initialHeight = content.offsetHeight;
			this.resizeObserver = new ResizeObserver(() => {
				if (content.offsetHeight <= initialHeight) return;
				const prev = el.style.minHeight;
				el.style.minHeight = "";
				const naturalHeight = Math.round(el.getBoundingClientRect().height);
				el.style.minHeight = prev;
				if (!this.tagsMinHeight || naturalHeight >= this.tagsMinHeight) {
					settings.cardHeights[this.id] = naturalHeight;
					this.tagsMinHeight = null;
					this.resizeObserver?.disconnect();
					this.resizeObserver = null;
				}
			});
			this.resizeObserver.observe(content);
		});
	},
	unmounted() {
		this.resizeObserver?.disconnect();
	},
};
</script>

<style scoped>
.root {
	display: block;
	list-style-type: none;
	border-radius: 1rem;
	padding: 1rem 1.5rem;
}
.root--with-tags {
	min-height: 8rem;
}
.root--unconfigured {
	background: none;
	border: 1px solid var(--evcc-gray-50);
	transition: border-color var(--evcc-transition-fast) linear;
	order: 1; /* unconfigured tiles at the end of the list */
}
.root--unconfigured:hover {
	border-color: var(--evcc-default-text);
}
.root--unconfigured :deep(.value),
.root--unconfigured :deep(.label) {
	color: var(--evcc-gray) !important;
	font-weight: normal !important;
}
.icon:empty {
	display: none;
}
.divide {
	margin-left: -1.5rem;
	margin-right: -1.5rem;
}
</style>
