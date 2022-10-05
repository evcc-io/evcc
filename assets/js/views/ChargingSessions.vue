<template>
	<div class="container px-4">
		<header class="d-flex justify-content-between align-items-center py-3">
			<h1 class="mb-1 pt-1 d-flex text-nowrap">
				<router-link class="dropdown-item mx-2 me-2" to="/">
					<shopicon-bold-arrowback size="s" class="back"></shopicon-bold-arrowback>
				</router-link>
				{{ $t("sessions.title") }}
			</h1>
			<TopNavigation />
		</header>

		<div class="row">
			<main class="col-12">
				<div class="mb-4">
					<a
						class="btn btn-outline-secondary text-nowrap my-2"
						href="./api/sessions?format=csv"
						download="sessions.csv"
					>
						{{ $t("sessions.downloadCsv") }}
					</a>
				</div>

				<div v-for="group in sessionsByMonth" :key="group.month" class="mb-5">
					<div class="mx-2 d-flex align-items-baseline">
						<h2 class="me-4">
							<span class="d-block d-md-none">
								{{ formatGroupHeadline(group.month, true) }}
							</span>
							<span class="d-none d-md-block">
								{{ formatGroupHeadline(group.month, false) }}
							</span>
						</h2>
						<div class="large mb-2">{{ fmtKWh(totalKWh(group.sessions)) }}</div>
						<ul class="groups">
							<li v-for="(loadpoint, id) in loadpointsKWh(group.sessions)" :key="id">
								{{ loadpoint.name }}: {{ fmtKWh(loadpoint.energy) }}
							</li>
							<li v-for="(vehicle, id) in vehiclesKWh(group.sessions)" :key="id">
								{{ vehicle.name }}: {{ fmtKWh(vehicle.energy) }}
							</li>
						</ul>
					</div>
					<div class="table-responsive">
						<table class="table">
							<thead>
								<tr>
									<th scope="col">{{ $t("sessions.loadpoint") }}</th>
									<th scope="col">{{ $t("sessions.vehicle") }}</th>
									<th scope="col">{{ $t("sessions.energy") }}</th>
									<th scope="col">{{ $t("sessions.date") }}</th>
								</tr>
							</thead>
							<tbody>
								<tr v-for="(session, id) in group.sessions" :key="id">
									<td class="text-nowrap">
										{{ session.loadpoint || $t("main.loadpoint.fallbackName") }}
									</td>
									<td class="text-nowrap">
										{{ session.vehicle || $t("main.vehicle.unknown") }}
									</td>
									<td class="text-nowrap">
										{{ fmtKWh(session.chargedEnergy * 1e3) }}
									</td>
									<td class="text-nowrap">
										{{ fmtFullDateTime(new Date(session.finished)) }}
									</td>
								</tr>
							</tbody>
						</table>
					</div>
				</div>
			</main>
		</div>
	</div>
</template>

<script>
import TopNavigation from "../components/TopNavigation.vue";
import "@h2d2/shopicons/es/bold/arrowback";
import formatter from "../mixins/formatter";
import api from "../api";

export default {
	name: "ChargingSessions",
	components: { TopNavigation },
	mixins: [formatter],
	props: {
		notifications: Array,
	},
	data() {
		return { sessions: [] };
	},
	computed: {
		sessionsByMonth() {
			const grouped = this.sessions.reduce((groups, session) => {
				const date = new Date(session.finished);
				const month = `${date.getFullYear()}.${date.getMonth()}`;
				if (!groups[month]) groups[month] = [];
				groups[month].push(session);
				return groups;
			}, {});
			return Object.entries(grouped).map(([month, sessions]) => {
				return { month, sessions };
			});
		},
	},
	mounted() {
		this.loadSessions();
	},
	methods: {
		async loadSessions() {
			const response = await api.get("sessions");
			this.sessions = response.data?.result;
		},
		totalKWh(sessions) {
			return sessions.reduce((total, session) => total + session.chargedEnergy, 0) * 1e3;
		},
		vehiclesKWh(sessions) {
			const grouped = sessions.reduce((groups, session) => {
				const vehicle = session.vehicle || this.$t("main.vehicle.unknown");
				if (!groups[vehicle]) groups[vehicle] = 0;
				groups[vehicle] += session.chargedEnergy * 1e3;
				return groups;
			}, {});
			return Object.entries(grouped).map(([name, energy]) => {
				return { name, energy };
			});
		},
		loadpointsKWh(sessions) {
			const grouped = sessions.reduce((groups, session) => {
				const loadpoint = session.loadpoint || this.$t("main.loadpoint.fallbackName");
				if (!groups[loadpoint]) groups[loadpoint] = 0;
				groups[loadpoint] += session.chargedEnergy * 1e3;
				return groups;
			}, {});
			return Object.entries(grouped).map(([name, energy]) => {
				return { name, energy };
			});
		},
		formatGroupHeadline(group, short) {
			const date = new Date();
			const [year, month] = group.split(".");
			date.setMonth(month);
			date.setFullYear(year);
			return this.fmtMonthYear(date, short);
		},
	},
};
</script>
<style scoped>
.back {
	width: 22px;
	height: 22px;
	position: relative;
	top: -2px;
}
.groups {
	display: flex;
	flex-wrap: wrap;
	list-style: none;
}

.groups li {
	white-space: nowrap;
}

.groups li:after {
	content: ", ";
	white-space: wrap;
	margin-right: 0.25rem;
}
.groups li:last-child:after {
	content: "";
}
</style>
