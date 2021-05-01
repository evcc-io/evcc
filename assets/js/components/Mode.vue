<template>
	<div>
		<div class="mb-3">Modus</div>
		<div class="btn-group w-100" role="group">
			<input
				id="mode_off"
				type="radio"
				name="mode"
				class="btn-check"
				value="off"
				@click="setTargetMode('off')"
				:checked="mode == 'off'"
			/>
			<label for="mode_off" class="btn btn-outline-primary"> Stop </label>
			<input
				id="mode_now"
				type="radio"
				name="mode"
				class="btn-check"
				value="now"
				@click="setTargetMode('now')"
				:checked="mode == 'now'"
			/>
			<label for="mode_now" class="btn btn-outline-primary"> Sofort </label>
			<input
				id="mode_minpv"
				type="radio"
				name="mode"
				class="btn-check"
				value="minpv"
				@click="setTargetMode('minpv')"
				:checked="mode == 'minpv'"
			/>
			<label for="mode_minpv" class="btn btn-outline-primary">
				<span class="d-inline d-sm-none">Min</span>
				<span class="d-none d-sm-inline">Min + PV</span>
			</label>
			<input
				id="mode_pv"
				type="radio"
				name="mode"
				class="btn-check"
				value="pv"
				@click="setTargetMode('pv')"
				:checked="mode == 'pv'"
			/>
			<label for="mode_pv" class="btn btn-outline-primary">
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
