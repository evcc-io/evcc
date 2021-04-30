<template>
	<div>
		<div class="mb-3">Modus</div>
		<div class="btn-group w-100" role="group">
			<input
				id="mode_off"
				type="radio"
				class="btn-check"
				value="off"
				@click="setTargetMode('off')"
			/>
			<label
				for="mode_off"
				class="btn btn-outline-primary first"
				:class="{ active: mode == 'off' }"
			>
				Stop
			</label>
			<input
				id="mode_now"
				type="radio"
				class="btn-check"
				value="now"
				@click="setTargetMode('now')"
			/>
			<label
				for="mode_now"
				class="btn btn-outline-primary"
				:class="{ active: mode == 'now' }"
			>
				Sofort
			</label>
			<input
				id="mode_minpv"
				type="radio"
				class="btn-check"
				value="minpv"
				@click="setTargetMode('minpv')"
			/>
			<label
				for="mode_minpv"
				class="btn btn-outline-primary"
				:class="{ active: mode == 'minpv' }"
				v-if="pvConfigured"
			>
				<span class="d-inline d-sm-none">Min</span>
				<span class="d-none d-sm-inline">Min + PV</span>
			</label>
			<input
				id="mode_pv"
				type="radio"
				class="btn-check"
				value="pv"
				@click="setTargetMode('pv')"
			/>
			<label
				for="mode_pv"
				class="btn btn-outline-primary"
				:class="{ active: mode == 'pv' }"
				v-if="pvConfigured"
			>
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
		pvConfigured: Boolean,
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
