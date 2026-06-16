import { createServer, type Server } from "net";
import Aedes from "aedes";

// startBroker runs a local in-process MQTT broker for tests, replacing the
// public test.mosquitto.org dependency. It accepts the given credentials only.
export function startBroker(port: number, username: string, password: string): Promise<Server> {
  const aedes = new Aedes({
    authenticate(_client, u, p, done) {
      const ok = u === username && p?.toString() === password;
      done(ok ? null : Object.assign(new Error("auth failed"), { returnCode: 4 }), ok);
    },
  });

  const server = createServer(aedes.handle);
  // close the broker when the listener is closed
  server.on("close", () => aedes.close());

  return new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(port, () => resolve(server));
  });
}
