<script setup>
import VehicleIcon from "./VehicleIcon.vue";

const icons = [
	"car",
	"bike",
	"bus",
	"moped",
	"motorcycle",
	"scooter",
	"tractor",
	"shuttle",
	"van",
	"heater",
	"heatpump",
	"waterheater",
	"laundry",
	"dishwasher",
	"airpurifier",
	"climate",
	"cooler",
	"generic",
	"device",
	"meter",
	"compute",
	"tool",
];
</script>

<template>
	<Story :layout="{ type: 'grid', width: 150 }">
		<Variant v-for="icon in icons" :key="icon" :title="icon">
			<VehicleIcon :name="icon" size="xl" />
		</Variant>
		<Variant title="2 cars">
			<VehicleIcon :names="['car', 'car']" size="xl" />
		</Variant>
		<Variant title="car & 3 bikes">
			<VehicleIcon :names="['car', 'bike', 'bike', 'bike']" size="xl" />
		</Variant>
	</Story>
</template>
