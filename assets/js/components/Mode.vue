<template>
	<div>
		<div class="mb-3">Modus</div>
		<div class="btn-group btn-group-toggle bg-white shadow-none w-100">
			<label class="btn btn-outline-primary first" :class="{ active: mode == 'off' }">
				<input type="radio" value="off" @click="setTargetMode('off')" />Stop
			</label>
			<label class="btn btn-outline-primary" :class="{ active: mode == 'now' }">
				<input type="radio" value="now" @click="setTargetMode('now')" />Sofort
			</label>
			<label class="btn btn-outline-primary" :class="{ active: mode == 'minpv' }">
				<input type="radio" value="minpv" @click="setTargetMode('minpv')" />
				<span class="d-inline d-sm-none">Min</span>
				<span class="d-none d-sm-inline">Min + PV</span>
			</label>
			<label class="btn btn-outline-primary" :class="{ active: mode == 'pv' }">
				<input type="radio" value="pv" @click="setTargetMode('pv')" />
				<span class="d-inline d-sm-none">PV</span>
				<span class="d-none d-sm-inline">Nur PV</span>
			</label>
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
			this.$emit("updated", mode);
		},
	},
};
</script>

<style scoped>
.btn {
	/* equal width buttons */
	flex-basis: 0;
}
</style>
