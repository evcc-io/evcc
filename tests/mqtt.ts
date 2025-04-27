import mqtt from "mqtt";

export async function isMqttReachable(
	broker: string,
	username: string,
	password: string
): Promise<boolean> {
	try {
		const client = mqtt.connect(`mqtt://${broker}`, {
			connectTimeout: 2000,
			username,
			password,
		});

		await new Promise<void>((resolve, reject) => {
			client.once("connect", () => resolve());
			client.once("error", (err) => reject(err));
		});

		client.end();
		return true; // connection successful
	} catch {
		return false; // connection failed
	}
}
