<template>
	<div class="mode-group border d-inline-flex" role="group">
		<button
			v-for="m in modes"
			:key="m"
			type="button"
			class="btn flex-grow-1 flex-shrink-1"
			:class="{ active: isActive(m) }"
			@click="setTargetMode(m)"
		>
			{{ $t(`main.mode.${m}`) }}
		</button>
	</div>
</template>

<script>
export default {
	name: "Mode",
	props: {
		mode: String,
	},
	emits: ["updated"],
	data() {
		return {
			modes: ["off", "pv", "minpv", "now"],
		};
	},
	methods: {
		isActive: function (mode) {
			return this.mode === mode;
		},
		setTargetMode: function (mode) {
			this.$emit("updated", mode);
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
