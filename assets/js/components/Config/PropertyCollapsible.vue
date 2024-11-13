<template>
	<div v-if="$slots.advanced || $slots.more">
		<button
			class="btn btn-link btn-sm text-gray px-0 border-0 d-flex align-items-center mb-2"
			:class="open ? 'text-primary' : ''"
			type="button"
			@click="toggle"
		>
			<span v-if="open">{{ $t("config.general.hideAdvancedSettings") }}</span>
			<span v-else>{{ $t("config.general.showAdvancedSettings") }}</span>
			<DropdownIcon class="icon" :class="{ iconUp: open }" />
		</button>

		<Transition>
			<div v-if="open" class="pt-2">
				<slot name="advanced"></slot>
				<hr v-if="$slots.advanced && $slots.more" class="my-5" />
				<slot name="more"></slot>
			</div>
		</Transition>
	</div>
</template>

<script>
import DropdownIcon from "../MaterialIcon/Dropdown.vue";

export default {
	name: "PropertyCollapsible",
	components: { DropdownIcon },
	data() {
		return { open: false };
	},

	methods: {
		toggle() {
			this.open = !this.open;
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
.v-enter-active,
.v-leave-active {
	transition:
		transform var(--evcc-transition-medium) ease,
		opacity var(--evcc-transition-medium) ease;
	transform: translateY(0);
}

.v-enter-from,
.v-leave-to {
	opacity: 0;
	transform: translateY(-0.5rem);
}
</style>
