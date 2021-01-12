<template>
	<div class="card">
		<div class="card-header card-header-with-link">
			<h4 class="mb-0">Schnittstellen</h4>
		</div>
		<div class="card-body">
			<h5 class="card-title">MQTT</h5>
			<p class="card-text">nicht konfiguriert <a href="#">ändern</a></p>
			<h5 class="card-title">
				InfluxDB
				<span class="float-right text-right">
					<small class="text-success">verbunden</small>
				</span>
			</h5>
			<p class="card-text">https://influx.local/ <a href="#">ändern</a></p>
		</div>
	</div>
</template>

<script>
export default {
	name: "Interfaces",
};
</script>
<style scoped>
.card-header-with-link {
	display: flex;
	justify-content: space-between;
	align-items: baseline;
}
</style>
