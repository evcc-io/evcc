import type { DeviceType } from "@/types/evcc";
import api from "@/api";

export type Product = {
	group: string;
	name: string;
	template: string;
};

export type Template = {
	Params: TemplateParam[];
	Requirements: {
		Description: string;
	};
};

export type TemplateParam = {
	Name: string;
	Required: boolean;
	Advanced: boolean;
	Deprecated: boolean;
	Default: string | number | boolean | undefined;
	Choice?: string[];
};

export type ModbusCapability = "rs485" | "tcpip";

export type ModbusParam = TemplateParam & {
	ID?: string;
	Comset?: string;
	Baudrate?: number;
	Port?: number;
};

export type DeviceValues = {
	type: ConfigType;
	icon: string | undefined;
	deviceProduct: string | undefined;
	yaml: string | undefined;
	template: string | null;
	[key: string]: any;
};

export function handleError(e: any, msg: string) {
	console.error(e);
	let message = msg;
	const { error } = e.response.data || {};
	if (error) message += `: ${error}`;
	alert(message);
}

export const timeout = 15000;

export enum ConfigType {
	Template = "template",
	Custom = "custom",
}

export function applyDefaultsFromTemplate(template: Template | null, values: DeviceValues) {
	const params = template?.Params || [];
	params
		.filter((p) => p.Default && !values[p.Name])
		.forEach((p) => {
			values[p.Name] = p.Default;
		});
}

export function createDeviceUtils(deviceType: DeviceType) {
	function test(id: number | undefined, data: any) {
		let url = `config/test/${deviceType}`;
		if (id !== undefined) {
			url += `/merge/${id}`;
		}
		return api.post(url, data, { timeout });
	}

	function update(id: number, data: any) {
		return api.put(`config/devices/${deviceType}/${id}`, data);
	}

	function remove(id: number) {
		return api.delete(`config/devices/${deviceType}/${id}`);
	}

	async function load(id: number) {
		const response = await api.get(`config/devices/${deviceType}/${id}`);
		return response.data.result;
	}

	async function create(data: any) {
		const response = await api.post(`config/devices/${deviceType}`, data);
		return response.data.result;
	}

	async function loadProducts(lang?: string, usage?: string) {
		const params: Record<string, string | undefined> = { lang };
		if (usage) {
			params["usage"] = usage;
		}
		const response = await api.get(`config/products/${deviceType}`, { params });
		return response.data.result;
	}

	async function loadTemplate(templateName: string, lang?: string) {
		if (!templateName) return null;

		const opts = {
			params: {
				lang,
				name: templateName,
			},
		};
		const response = await api.get(`config/templates/${deviceType}`, opts);
		return response.data.result;
	}

	return {
		test,
		update,
		remove,
		load,
		create,
		loadProducts,
		loadTemplate,
	};
}
