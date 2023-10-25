<script setup>
import { reactive } from "vue";
import Notifications from "./Notifications.vue";

const timeAgo = (hours = 0, minutes = 0, seconds = 0) => {
	const date = new Date();
	date.setHours(date.getHours() - hours);
	date.setMinutes(date.getMinutes() - minutes);
	date.setSeconds(date.getSeconds() - seconds);
	return date;
};

const state = reactive({
	notifications: [
		{
			message: "Server unavailable",
			level: "error",
			time: timeAgo(),
			count: 1,
		},
		{
			message: "charger out of sync: expected disabled, got enabled",
			level: "warn",
			count: 4,
			time: timeAgo(0, 0, 42),
		},
		{
			message: "Sponsortoken: x509: certificate has expired",
			level: "error",
			count: 1,
			time: timeAgo(1, 12, 44),
		},
		{
			message:
				"Block device full: can not write to /tmp/evcc/foobarloremawefhwuiehfwuiehfwiauhefjkajowaeigjwalvmoweivwail",
			level: "error",
			count: 1,
			time: timeAgo(1, 12, 44),
		},
		{
			message: "charger out of sync: expected disabled, got enabled",
			level: "warn",
			count: 4,
			time: timeAgo(1, 22, 0),
		},
		{
			message:
				"vehicle remote charge start: invalid character '<' looking for beginning of value",
			level: "warn",
			count: 3,
			time: timeAgo(4, 2, 0),
		},
		{
			message:
				"Amet irure quis incididunt voluptate esse. Commodo ea sunt est ipsum tempor nisi laboris voluptate labore elit laborum. Ex irure commodo reprehenderit consequat consequat do ad tempor aliquip deserunt eu. Laboris minim nostrud quis nisi. Dolor occaecat reprehenderit velit dolore exercitation cupidatat et voluptate. Nulla pariatur deserunt esse minim nisi nisi nulla. Sit eiusmod do incididunt sint minim pariatur aute.",
			level: "warn",
			count: 1,
			time: timeAgo(5, 2, 44),
		},
	],
});
</script>

<template>
	<Story>
		<Variant title="default">
			<Notifications v-bind="state" />
		</Variant>
	</Story>
</template>
