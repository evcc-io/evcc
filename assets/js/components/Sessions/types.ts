export interface Session {
	id: number;
	created: string;
	finished: string;
	loadpoint: string;
	identifier: string;
	vehicle: string;
	odometer: number;
	meterStart: number;
	meterStop: number;
	chargedEnergy: number;
	chargeDuration: number;
	solarPercentage: number;
	price: number | null;
	pricePerKWh: number | null;
	co2PerKWh?: number | null;
}

export interface Legend {
	label: string;
	color: any;
	value: string | string[];
}

export interface Column {
	name: string;
	unit: string;
	total: number | null;
	value: (session: Session) => number | null;
	format: (value: number) => string | undefined;
}

export enum TYPES {
	SOLAR = "solar",
	PRICE = "price",
	CO2 = "co2",
}

export enum GROUPS {
	NONE = "none",
	LOADPOINT = "loadpoint",
	VEHICLE = "vehicle",
}

export enum PERIODS {
	MONTH = "month",
	YEAR = "year",
	TOTAL = "total",
}
