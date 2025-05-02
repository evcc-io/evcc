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
