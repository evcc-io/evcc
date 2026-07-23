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
			<hr class="my-3 divide" :class="{ 'border-warning': banner }" />
			<div
				v-if="banner"
				class="limit-stripe banner fw-bold text-center text-warning"
				data-testid="device-banner"
			>
				{{ banner }}
			</div>
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
		banner: String,
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
			const minContentHeight = 10;
			const check = () => {
				if (content.offsetHeight <= minContentHeight) return;
				// measure natural height without cached min-height
				el.style.minHeight = "";
				settings.cardHeights[this.id] = Math.round(el.getBoundingClientRect().height);
				this.tagsMinHeight = null;
				this.resizeObserver?.disconnect();
				this.resizeObserver = null;
			};
			this.resizeObserver = new ResizeObserver(check);
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
/* bleed edge-to-edge over the card padding, flush below the divider */
.banner {
	margin: -1rem -1.5rem 1rem;
	padding: 0.5rem 1.5rem;
}
/* animated throttle stripe, echoes the charging bar */
.limit-stripe {
	background-color: color-mix(in srgb, var(--evcc-orange) 9%, transparent);
	background-image: repeating-linear-gradient(
		-45deg,
		color-mix(in srgb, var(--evcc-orange) 20%, transparent) 0 8px,
		transparent 8px 20px
	);
	background-size: 28.28px 28.28px;
}

@media (prefers-reduced-motion: no-preference) {
	.limit-stripe {
		animation: limit-stripe-move 1.5s linear infinite;
	}
}

@keyframes limit-stripe-move {
	to {
		background-position: 28.28px 0;
	}
}
</style>
