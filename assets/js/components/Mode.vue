<template>
	<div v-if="$hiddenFeatures" class="mode-group border d-inline-flex" role="group">
		<button
			v-for="m in ['fast', 'cheap']"
			:key="m"
			type="button"
			class="btn"
			:class="{ active: isActive(m) }"
			@click="setTargetMode(m)"
		>
			{{ $t(`main.mode.${m}`) }}
		</button>
	</div>
	<div v-else class="mode-group border d-inline-flex" role="group">
		<button
			v-for="m in modes"
			:key="m"
			type="button"
			class="btn"
			:class="{ active: isActive(m) }"
			@click="setTargetMode(m)"
		>
			<span class="d-inline d-sm-none"> {{ $t(`main.mode.${m}Short`) }} </span>
			<span class="d-none d-sm-inline"> {{ $t(`main.mode.${m}Long`) }} </span>
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
			modes: ["off", "now", "minpv", "pv"],
		};
	},
	methods: {
		isActive: function (mode) {
			return this.mode === this.mapToOldModes(mode);
		},
		setTargetMode: function (mode) {
			this.$emit("updated", this.mapToOldModes(mode));
		},
		mapToOldModes: function (mode) {
			const mapping = { fast: "now", cheap: "pv" };
			return mapping[mode] || mode;
		},
	},
};
</script>

<style scoped></style>

<style scoped>
.mode-group {
	border: 2px solid var(--bs-gray-medium);
	background-color: var(--bs-white);
	border-radius: 20px;
	padding: 4px;
}

.btn {
	/* equal width buttons */
	flex-basis: 0;
	white-space: nowrap;
	border-radius: 18px;
	padding: 0.1em 0.8em;
}
.btn.active {
	background: var(--bs-gray-dark);
	color: var(--bs-white);
}
.btn-group {
	border-radius: 16px;
}
</style>
