import { startBroker } from "./mqtt-broker";
import { LOCAL_BROKER_PORT, LOCAL_USERNAME, LOCAL_PASSWORD } from "./mqtt";

// globalSetup starts a single local MQTT broker shared by all workers and returns
// a teardown that stops it. Playwright runs the returned function after the suite.
export default async function globalSetup() {
  const server = await startBroker(LOCAL_BROKER_PORT, LOCAL_USERNAME, LOCAL_PASSWORD);
  return () => new Promise<void>((resolve) => server.close(() => resolve()));
}
