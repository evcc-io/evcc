<template>
	<div class="mode-group border d-inline-flex" :class="{ disabled: !mode }" role="group">
		<button
			type="button"
			class="btn flex-grow-1 flex-shrink-1"
			:class="{ active: isActive('aus') }"
		>
			aus
		</button>
		<button
			type="button"
			class="btn flex-grow-1 flex-shrink-1"
			:class="{ active: isActive('an') }"
		>
			an
		</button>
		<button
			type="button"
			class="btn flex-grow-1 flex-shrink-1"
			:class="{ active: isActive('smart') }"
		>
			smart
		</button>
	</div>
</template>

<script>
export default {
	name: "MiniMode",
	props: {
		mode: String,
	},
	methods: {
		isActive: function (mode) {
			return this.mode === mode;
		},
	},
};
</script>

<style scoped>
.mode-group {
	border: 2px solid var(--evcc-default-text);
	border-radius: 20px;
	padding: 4px;
	min-width: 255px;
}

.disabled button {
	opacity: 0.2;
}

.btn {
	/* equal width buttons */
	flex-basis: 0;
	white-space: nowrap;
	border-radius: 18px;
	padding: 0.1em 0.8em;
	color: var(--evcc-default-text);
	border: none;
}
.btn:hover {
	color: var(--evcc-gray);
}
.btn.active {
	color: var(--evcc-background);
	background: var(--evcc-default-text);
}
.btn-group {
	border-radius: 16px;
}
</style>
