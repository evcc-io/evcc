<template>
	<div>
		<div class="my-4">
			<h2 style="display: inline-block" class="pr-3">Ladepunkte</h2>
			<a href="#">hinzufügen</a>
		</div>

		<div class="card-deck mt-4 mb-5">
			<div class="card">
				<div class="card-header card-header-with-link">
					<h4 class="mb-0">Carport</h4>
					<a class="text" href="#">umbenennen</a>
				</div>
				<div class="card-body">
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Wallbox</h5>
						&nbsp;
						<a href="#">ändern</a>
						<span class="float-right text-right">
							<h5 class="mb-0">3,7 kW</h5>
							<small class="text-muted">lädt</small>
						</span>
					</div>
					<p class="card-text">NRGKick Connect</p>
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Fahrzeuge</h5>
						&nbsp;
						<a href="#">ändern</a>
					</div>
					<p class="card-text">VW ID.3</p>
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Ladeverhalten</h5>
						&nbsp;
						<a href="#">ändern</a>
					</div>
					<p class="card-text">
						Modus: PV-only<br />
						Ladeziel: 90% (sofort 20%)<br />
						Ladeleistung: 6A bis 16A, 3-phasig<br />
					</p>
				</div>
				<div class="card-footer text-right">
					<a class="text-danger" href="#">entfernen</a>
				</div>
			</div>
			<div class="card">
				<div class="card-header card-header-with-link">
					<h4 class="mb-0">Garage</h4>
					<a class="text" href="#">umbenennen</a>
				</div>
				<div class="card-body">
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Wallbox</h5>
						&nbsp;
						<a href="#">ändern</a>
						<span class="float-right text-right">
							<h5 class="mb-0">0 kW</h5>
							<small class="text-muted">getrennt</small>
						</span>
					</div>
					<p class="card-text">KEBA X30</p>
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Fahrzeuge</h5>
						&nbsp;
						<a href="#">ändern</a>
					</div>
					<p class="card-text">
						Tesla Model 3<br />
						VW ID.3
					</p>
					<div class="card-title">
						<h5 class="mb-0" style="display: inline-block">Ladeverhalten</h5>
						&nbsp;
						<a href="#">ändern</a>
					</div>
					<p class="card-text">
						Modus: schnell<br />
						Ladeziel: 90%<br />
						Ladeleistung: 6A bis 32A, 3-phasig<br />
					</p>
				</div>
				<div class="card-footer text-right">
					<a class="text-danger" href="#">entfernen</a>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
export default {
	name: "Loadpoints",
};
</script>
<style scoped>
.card-header-with-link {
	display: flex;
	justify-content: space-between;
	align-items: baseline;
}
</style>
