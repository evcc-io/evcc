<template>
	<div class="container">
		<div class="row mt-4">
			<div class="col-12">
				<h1>Setup</h1>
			</div>
		</div>
		<div class="row mt-4">
			<div class="col-6">
				<div class="card">
					<div class="card-header card-header-with-link">
						<h3 class="mb-0">Haus</h3>
						<a class="text" href="#">umbenennen</a>
					</div>
					<div class="card-body">
						<h5 class="card-title">
							PV Anlage
							<span class="float-right text-success">5,42 kW</span>
						</h5>
						<p class="card-text">SMA Sunny Tripower 8.0 <a href="#">ändern</a></p>
						<h5 class="card-title">
							Hausverbrauch
							<span class="float-right">4,20 kW</span>
						</h5>
						<p class="card-text">abgeleitet <a href="#">ändern</a></p>
						<h5 class="card-title">
							Hausbatterie
							<span class="float-right text-right">
								<span class="text-success">+1,22 kW</span><br />
								<small class="text-muted">76%</small>
							</span>
						</h5>
						<p class="card-text">BYD B-BOX PREMIUM 9.0kWh <a href="#">ändern</a></p>
						<h5 class="card-title">
							Netzanschluss
							<span class="float-right">0,00 kW</span>
						</h5>
						<p class="card-text">Discovergy Zähler <a href="#">ändern</a></p>
					</div>
				</div>
			</div>
			<div class="col-6">
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
			</div>
		</div>
	</div>
</template>

<script>
export default {
	name: "Setup",
	data: function () {
		return this.$root.$data.store.state; // global state
	},
};
</script>
<style scoped>
.card-header-with-link {
	display: flex;
	justify-content: space-between;
	align-items: baseline;
}
</style>
