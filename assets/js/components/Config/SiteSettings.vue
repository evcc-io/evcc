<template>
	<div class="group pt-4 px-4 pb-1">
		<dl class="row">
			<dt class="col-sm-4">Title</dt>
			<dd class="col-sm-8">
				{{ title }}
				<a href="#" class="ms-2 d-inline-block text-muted" @click.prevent="todo">edit</a>
			</dd>
		</dl>
		<dl class="row wip">
			<dt class="col-sm-4">Password</dt>
			<dd class="col-sm-8">
				*******
				<a href="#" class="ms-2 d-inline-block text-muted" @click.prevent="todo">edit</a>
			</dd>
		</dl>
		<dl class="row wip">
			<dt class="col-sm-4">API-Key</dt>
			<dd class="col-sm-8">
				*******
				<a href="#" class="ms-2 d-inline-block text-muted" @click.prevent="todo">show</a>
			</dd>
		</dl>
		<dl class="row wip">
			<dt class="col-sm-4">Sponsoring</dt>
			<dd class="col-sm-8">
				<span class="text-primary"> valid </span>
				<a href="#" class="ms-2 d-inline-block text-muted" @click.prevent="todo">change</a>
			</dd>
		</dl>
		<dl class="row wip">
			<dt class="col-sm-4">Telemetry</dt>
			<dd class="col-sm-8">
				on
				<a href="#" class="ms-2 d-inline-block text-muted" @click.prevent="todo">toggle</a>
			</dd>
		</dl>
		<dl class="row wip">
			<dt class="col-sm-4">Server</dt>
			<dd class="col-sm-8">
				http://evcc.local:7070
				<a href="#" class="ms-2 d-inline-block text-muted" @click.prevent="todo">edit</a>
			</dd>
		</dl>
		<dl class="row wip">
			<dt class="col-sm-4">Update Interval</dt>
			<dd class="col-sm-8">
				30s
				<a href="#" class="ms-2 d-inline-block text-muted" @click.prevent="todo">edit</a>
			</dd>
		</dl>
	</div>
</template>

<script>
import api from "../../api";

export default {
	name: "SiteSettings",
	data() {
		return {
			title: "",
		};
	},
	emits: ["site-changed"],
	async mounted() {
		await this.load();
	},
	methods: {
		async load() {
			try {
				const { data } = await api.get("/config/site");
				this.title = data.result.title;
			} catch (e) {
				console.error(e);
			}
		},
		todo() {
			alert("not implemented");
		},
	},
};
</script>

<style scoped>
.group {
	border-radius: 1rem;
	box-shadow: 0 0 0 0 var(--evcc-gray-50);
	color: var(--evcc-default-text);
	background: var(--evcc-box);
	padding: 1rem;
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
	min-height: 10rem;
	margin-bottom: 5rem;
	border: 1px solid var(--evcc-gray-50);
	transition: box-shadow var(--evcc-transition-fast) linear;
}

.group:hover {
	border-color: var(--evcc-gray);
}

.group:focus-within {
	box-shadow: 0 0 1rem 0 var(--evcc-gray-50);
}

.wip {
	opacity: 0.2;
}
dt {
	margin-bottom: 0.5rem;
}
dd {
	margin-bottom: 1rem;
}
</style>
