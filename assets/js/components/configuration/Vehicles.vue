<template>
	<div class="card">
		<div class="card-header card-header-with-link">
			<h3 class="mb-0">Fahrzeuge</h3>
			<a class="text" href="#">hinzufügen</a>
		</div>
		<div class="card-body">
			<h5 class="card-title">
				VW ID.3
				<span class="float-right text-right">
					76 %<br />
					<small class="text-muted">lädt 3,7 kW</small>
				</span>
			</h5>
			<p class="card-text">Volkswagen Connect <a href="#">ändern</a></p>
			<h5 class="card-title">
				Tesla Model 3
				<span class="float-right text-right">
					33 %<br />
					<small class="text-muted">schläft</small>
				</span>
			</h5>
			<p class="card-text">Tesla API <a href="#">ändern</a></p>
		</div>
	</div>
</template>

<script>
export default {
	name: "Vehicles",
};
</script>
<style scoped>
.card-header-with-link {
	display: flex;
	justify-content: space-between;
	align-items: baseline;
}
</style>
