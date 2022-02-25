<template>
	<div>
		<div class="mode-group border d-inline-flex" role="group">
			<button
				v-for="m in modes"
				:key="m"
				type="button"
				class="btn"
				:class="{ active: mode == m }"
				@click="setTargetMode(m)"
			>
				<span class="d-inline d-sm-none"> {{ $t(`main.mode.${m}Short`) }} </span>
				<span class="d-none d-sm-inline"> {{ $t(`main.mode.${m}Long`) }} </span>
			</button>
		</div>
	</div>
</template>

<script>
export default {
	name: "Mode",
	props: {
		mode: String,
	},
	data() {
		return {
			modes: ["off", "now", "minpv", "pv"],
		};
	},
	methods: {
		setTargetMode: function (mode) {
			this.$emit("updated", mode);
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
