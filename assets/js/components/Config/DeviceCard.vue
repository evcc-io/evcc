<template>
	<li class="root">
		<div class="d-flex align-items-center mb-2">
			<div class="icon me-2">
				<slot name="icon" />
			</div>
			<span class="flex-grow-1 text-nowrap text-truncate">{{ name }}</span>
			<button
				v-if="unconfigured"
				class="btn btn-sm btn-link text-gray"
				@click="$emit('configure')"
			>
				{{ $t("config.main.unconfigured") }}
			</button>
			<button
				v-else-if="editable"
				class="btn btn-sm btn-link text-gray"
				@click="$emit('edit')"
			>
				{{ $t("config.main.edit") }}
			</button>
			<span v-else class="text-gray opacity-50 px-2 py-1" disabled>yaml</span>
		</div>
		<div v-if="tags" class="d-flex mb-2" @click="todo">
			<span
				v-for="(tag, index) in tags"
				:key="index"
				class="badge text-bg-secondary me-1 mb-1"
			>
				{{ tag }}
			</span>
		</div>
	</li>
</template>

<script>
export default {
	name: "DeviceCard",
	props: {
		name: String,
		editable: Boolean,
		unconfigured: Boolean,
		tags: Array,
	},
	emits: ["edit", "configure"],
};
</script>

<style scoped>
.root {
	border-radius: 1rem;
	color: var(--evcc-default-text);
	background: var(--evcc-box);
	padding: 1rem 1rem 0.5rem;
	display: block;
	list-style-type: none;
}
.icon:empty {
	display: none;
}
</style>
