import { WebSocketServer, WebSocket } from "ws";
import type { IncomingMessage } from "http";
import type { Duplex } from "stream";

// minimal surface we need from the shared http(s)/http2 server
type UpgradableServer = {
  on(
    event: "upgrade",
    listener: (req: IncomingMessage, socket: Duplex, head: Buffer) => void
  ): unknown;
};

// OcppServer is a minimal upstream OCPP server used to test the evcc forwarder.
// It is NOT spec compliant: it accepts a WebSocket connection (optionally behind
// HTTP Basic Auth) and answers Calls with canned CallResults so the connection
// stays alive. It shares the simulator's HTTP port via the vite httpServer.
class OcppServer {
  // echo the ocpp1.6 subprotocol so clients that require negotiation (evcc's
  // coder/websocket dialer) complete the handshake
  private wss = new WebSocketServer({ noServer: true, handleProtocols: () => "ocpp1.6" });
  private sockets = new Set<WebSocket>();

  enabled = false;
  username = "";
  password = "";
  lastStationId: string | null = null;

  // attach hooks the WebSocket upgrade on the shared http server. OCPP upgrades
  // are recognised by the "ocpp1.6" subprotocol; everything else (e.g. vite HMR)
  // is left for other listeners.
  attach(httpServer: UpgradableServer) {
    httpServer.on("upgrade", (req: IncomingMessage, socket: Duplex, head: Buffer) => {
      const protocols = String(req.headers["sec-websocket-protocol"] || "");
      if (!protocols.includes("ocpp1.6")) return; // not an OCPP upgrade (e.g. vite HMR)

      // server off: reject the OCPP upgrade so the client errors out and retries,
      // rather than leaving the handshake hanging with no response
      if (!this.enabled) {
        socket.destroy();
        return;
      }

      if ((this.username || this.password) && !this.checkAuth(req)) {
        console.log("[ocpp-server] rejected: bad credentials");
        socket.write('HTTP/1.1 401 Unauthorized\r\nWWW-Authenticate: Basic realm="ocpp"\r\n\r\n');
        socket.destroy();
        return;
      }

      this.wss.handleUpgrade(req, socket, head, (ws) => this.onConnection(ws, req));
    });
  }

  configure(opts: { enabled: boolean; username?: string; password?: string }) {
    this.enabled = opts.enabled;
    this.username = opts.username || "";
    this.password = opts.password || "";
    if (!this.enabled) this.closeAll();
  }

  status() {
    return {
      enabled: this.enabled,
      username: this.username,
      password: this.password,
      lastStationId: this.lastStationId,
      connections: this.sockets.size,
    };
  }

  private checkAuth(req: IncomingMessage): boolean {
    const header = String(req.headers["authorization"] || "");
    const expected = "Basic " + Buffer.from(`${this.username}:${this.password}`).toString("base64");
    return header === expected;
  }

  private onConnection(ws: WebSocket, req: IncomingMessage) {
    const path = (req.url || "/").split("?")[0];
    const stationId = decodeURIComponent(path.replace(/^\/+/, "")) || "(root)";
    this.lastStationId = stationId;
    this.sockets.add(ws);
    console.log(`[ocpp-server] ${stationId} connected (${this.sockets.size} active)`);

    ws.on("message", (data) => this.handleMessage(ws, data.toString()));
    ws.on("close", () => {
      this.sockets.delete(ws);
      console.log(`[ocpp-server] ${stationId} disconnected (${this.sockets.size} active)`);
    });
  }

  // answer charger Calls (messageType 2) with a CallResult (messageType 3)
  private handleMessage(ws: WebSocket, raw: string) {
    let msg: unknown;
    try {
      msg = JSON.parse(raw);
    } catch {
      return;
    }
    if (!Array.isArray(msg) || msg[0] !== 2) return; // only respond to Calls
    const [, messageId, action] = msg as [number, string, string, unknown];
    ws.send(JSON.stringify([3, messageId, this.resultFor(action)]));
  }

  private resultFor(action: string): Record<string, unknown> {
    const now = new Date().toISOString();
    switch (action) {
      case "BootNotification":
        return { status: "Accepted", currentTime: now, interval: 300 };
      case "Heartbeat":
        return { currentTime: now };
      case "Authorize":
        return { idTagInfo: { status: "Accepted" } };
      case "StartTransaction":
        return { transactionId: 1, idTagInfo: { status: "Accepted" } };
      case "StopTransaction":
        return { idTagInfo: { status: "Accepted" } };
      default:
        return {}; // StatusNotification, MeterValues, DataTransfer, ...
    }
  }

  private closeAll() {
    for (const ws of this.sockets) ws.close();
    this.sockets.clear();
  }
}

export const ocppServer = new OcppServer();
