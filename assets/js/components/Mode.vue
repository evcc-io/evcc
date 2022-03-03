<template>
	<div>
		<div class="mb-3">{{ $t("main.mode.title") }}</div>
		<div class="btn-group w-100" role="group">
			<button
				type="button"
				class="btn btn-outline-secondary"
				:class="{ active: mode == 'off' }"
				@click="setTargetMode('off')"
			>
				{{ $t("main.mode.stop") }}
			</button>
			<button
				type="button"
				class="btn btn-outline-secondary"
				:class="{ active: mode == 'now' }"
				@click="setTargetMode('now')"
			>
				{{ $t("main.mode.now") }}
			</button>
			<button
				type="button"
				class="btn btn-outline-secondary"
				:class="{ active: mode == 'minpv' }"
				@click="setTargetMode('minpv')"
			>
				<span class="d-inline d-sm-none"> {{ $t("main.mode.minpvShort") }} </span>
				<span class="d-none d-sm-inline"> {{ $t("main.mode.minpvLong") }} </span>
			</button>
			<button
				type="button"
				class="btn btn-outline-secondary"
				:class="{ active: mode == 'pv' }"
				@click="setTargetMode('pv')"
			>
				<span class="d-inline d-sm-none"> {{ $t("main.mode.pvShort") }} </span>
				<span class="d-none d-sm-inline"> {{ $t("main.mode.pvLong") }} </span>
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
	methods: {
		setTargetMode: function (mode) {
			this.$emit("update", mode);
		},
	},
};
</script>

<style scoped>
.btn {
	/* equal width buttons */
	flex-basis: 0;
	white-space: nowrap;
}
</style>
