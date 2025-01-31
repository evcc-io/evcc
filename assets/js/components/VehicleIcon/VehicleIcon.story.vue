<script setup>
import VehicleIcon from "./VehicleIcon.vue";

const icons = [
	"car",
	"bike",
	"bus",
	"moped",
	"motorcycle",
	"rocket",
	"scooter",
	"taxi",
	"tractor",
	"rickshaw",
	"shuttle",
	"van",
	"airpurifier",
	"battery",
	"bulb",
	"climate",
	"coffeemaker",
	"compute",
	"cooking",
	"cooler",
	"desktop",
	"device",
	"dishwasher",
	"dryer",
	"floorlamp",
	"generic",
	"heater",
	"heatexchange",
	"heatpump",
	"kettle",
	"laundry",
	"laundry2",
	"machine",
	"meter",
	"microwave",
	"pump",
	"tool",
	"waterheater",
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
