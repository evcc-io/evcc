export interface Notification {
	message: string;
	time: Date;
	level: string;
	lp: number;
	count: number;
}

export interface Provider {
	title: string;
	loggedIn: boolean;
	loginPath: string;
	logoutPath: string;
}
