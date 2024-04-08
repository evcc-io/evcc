<template>
	<div class="root safe-area-inset">
		<div class="container d-flex h-100 flex-column px-4 pb-5">
			<TopHeader title="Log" />
			<div class="logs d-flex flex-column overflow-hidden flex-grow-1 px-4">
				<div
					class="flex-grow-0 d-flex py-4 justify-content-between gap-4 flex-wrap flex-md-nowrap"
				>
					<div class="autoFollow order-1">
						<button
							class="btn btn-sm text-nowrap"
							:class="autoFollow ? 'btn-primary' : 'btn-outline-secondary'"
							type="button"
							@click="toggleAutoFollow"
						>
							<span v-if="autoFollow">Disable auto-follow</span>
							<span v-else>Enable auto-follow</span>
						</button>
					</div>
					<input
						type="search"
						class="form-control form-control-sm order-3 order-md-2 search"
						placeholder="Filter"
						v-model="search"
					/>
					<div class="logLevelFilter order-2 order-md-3 d-flex justify-content-end">
						<div
							class="btn-group btn-group-sm text-nowrap"
							role="group"
							aria-label="Basic radio toggle button group"
						>
							<template
								v-for="level in ['trace', 'debug', 'warn', 'error']"
								:key="level"
							>
								<input
									type="radio"
									class="btn-check"
									name="logLevel"
									:id="`logLevel-${level}`"
									:value="level"
									v-model="logLevel"
								/>
								<label class="btn btn-outline-secondary" :for="`logLevel-${level}`">
									{{ level }}
								</label>
							</template>
						</div>
					</div>
				</div>
				<hr class="my-0" />
				<code class="d-block overflow-y-scroll" ref="log" @scroll="onScroll">
					<div v-for="(line, index) in filteredLines" :key="index">
						{{ line }}
					</div>
				</code>
			</div>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/backdown";
import TopHeader from "../components/TopHeader.vue";
import api from "../api";

export default {
	name: "Log",
	components: {
		TopHeader,
	},
	data() {
		return {
			lines: [],
			autoFollow: false,
			logLevel: "debug",
			search: "",
		};
	},
	mounted() {
		this.load();
	},
	computed: {
		filteredLines() {
			return this.lines
				.filter((line) => !this.search || line.includes(this.search))
				.filter((line) => line.includes(this.logLevel.toUpperCase()));
		},
	},
	methods: {
		async load() {
			try {
				const response = await api.get("/log");
				this.lines = response.data?.result;
			} catch (e) {
				console.error(e);
			}
		},
		onScroll() {
			// enable auto-follow if scrolled to bottom, else set it to false
			this.autoFollow =
				this.$refs.log.scrollHeight - this.$refs.log.scrollTop ===
				this.$refs.log.clientHeight;
		},
		toggleAutoFollow() {
			this.autoFollow = !this.autoFollow;
			if (this.autoFollow) {
				this.$refs.log.scrollTop = this.$refs.log.scrollHeight;
			}
		},
	},
};
</script>
<style scoped>
.logs {
	border-radius: 2rem;
	background: var(--evcc-box);
}
.root {
	height: 100vh;
	height: 100dvh;
}
.btn {
	--bs-btn-border-width: 1px;
}
</style>
