export interface Grid {
	power: number;
}

export interface Loadpoint {
	vehicleName: string;
	chargerIcon?: string;
	vehicleSoc: number;
	title?: string;
	charging: boolean;
	chargePower?: number;
	chargerFeatureHeating: boolean;
	icon: string;
	index: number;
}
