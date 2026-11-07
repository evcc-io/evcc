<template>
	<div v-if="$slots.advanced || $slots.more">
		<button
			class="btn btn-link btn-sm text-gray px-0 border-0 d-flex align-items-center mb-2"
			:class="open ? 'text-primary' : ''"
			type="button"
			tabindex="0"
			@click="toggle"
		>
			<span v-if="open">{{ $t("config.general.hideAdvancedSettings") }}</span>
			<span v-else>{{ $t("config.general.showAdvancedSettings") }}</span>
			<DropdownIcon class="icon" :class="{ iconUp: open }" />
		</button>

		<div class="collapsible-wrapper" :class="{ open }" @transitionend="onTransitionEnd">
			<div class="collapsible-content ring-space">
				<div v-if="rendered" class="pt-2">
					<slot name="advanced"></slot>
					<hr v-if="$slots.advanced && $slots.more" class="my-5" />
					<slot name="more"></slot>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import DropdownIcon from "../MaterialIcon/Dropdown.vue";

export default {
	name: "PropertyCollapsible",
	components: { DropdownIcon },
	props: {
		expanded: Boolean,
	},
	data() {
		return { open: this.expanded, rendered: this.expanded };
	},
	watch: {
		expanded(value) {
			this.open = value;
		},
		open(value) {
			if (value) this.rendered = true;
		},
	},

	methods: {
		toggle() {
			this.open = !this.open;
		},
		// unmount after collapse so hidden fields don't stay in the form
		onTransitionEnd(e) {
			if (e.target === e.currentTarget && !this.open) this.rendered = false;
		},
	},
};
</script>
<style scoped>
.icon {
	transform: rotate(0deg);
	transition: transform var(--evcc-transition-medium) ease;
}
.iconUp {
	transform: rotate(-180deg);
}
</style>
