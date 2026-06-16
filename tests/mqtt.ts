import mqtt from "mqtt";

// local broker started in globalSetup (see tests/global-setup.ts), replacing the
// flaky public test.mosquitto.org dependency
export const LOCAL_BROKER_PORT = 18831;
export const LOCAL_BROKER = `localhost:${LOCAL_BROKER_PORT}`;
export const LOCAL_USERNAME = "rw";
export const LOCAL_PASSWORD = "readwrite";

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
