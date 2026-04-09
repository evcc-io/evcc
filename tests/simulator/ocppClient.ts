import WebSocket from "ws";

export class OcppClient {
  private ws: WebSocket | null = null;
  private messageId = 0;
  private pendingCallbacks = new Map<string, (response: any) => void>();
  private connected = false;
  private stationId: string;
  private serverUrl: string;

  constructor(stationId: string, serverUrl: string) {
    this.stationId = stationId;
    this.serverUrl = serverUrl;
  }

  async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const url = `${this.serverUrl}${this.stationId}`;
      console.log(`[OCPP Client] Connecting to ${url}`);

      this.ws = new WebSocket(url, ["ocpp1.6"]);

      this.ws.on("open", () => {
        console.log(`[OCPP Client] Connected as ${this.stationId}`);
        this.connected = true;
        resolve();
      });

      this.ws.on("message", (data: WebSocket.Data) => {
        const message = JSON.parse(data.toString());
        this.handleMessage(message);
      });

      this.ws.on("error", (error) => {
        console.error(`[OCPP Client] Error:`, error);
        reject(error);
      });

      this.ws.on("close", () => {
        console.log(`[OCPP Client] Disconnected`);
        this.connected = false;
      });

      // Timeout after 5 seconds
      setTimeout(() => {
        if (!this.connected) {
          this.disconnect();
          reject(new Error("Connection timeout"));
        }
      }, 5000);
    });
  }

  private handleMessage(message: any[]) {
    const messageType = message[0];

    // Handle CallResult (3) - response to our calls
    if (messageType === 3) {
      const messageId = message[1];
      const payload = message[2];
      const callback = this.pendingCallbacks.get(messageId);
      if (callback) {
        callback(payload);
        this.pendingCallbacks.delete(messageId);
      }
    }
    // Handle Call (2) - requests from server
    else if (messageType === 2) {
      const messageId = message[1];
      const action = message[2];
      const payload = message[3];
      console.log(`[OCPP Client] Received ${action}:`, payload);

      // Auto-respond to common server requests
      this.handleServerRequest(messageId, action, payload);
    }
  }

  private handleServerRequest(messageId: string, action: string, payload: any) {
    let response: any = {};

    switch (action) {
      case "GetConfiguration":
        response = {
          configurationKey: [
            { key: "NumberOfConnectors", readonly: true, value: "1" },
            {
              key: "MeterValuesSampledData",
              readonly: false,
              value: "Energy.Active.Import.Register,Power.Active.Import",
            },
          ],
        };
        break;
      case "ChangeAvailability":
        response = { status: "Accepted" };
        break;
      case "RemoteStartTransaction":
        response = { status: "Accepted" };
        break;
      case "RemoteStopTransaction":
        response = { status: "Accepted" };
        break;
      case "TriggerMessage":
        // Handle trigger and send the requested message
        if (payload.requestedMessage === "StatusNotification") {
          this.statusNotification(1, "Available");
        }
        response = { status: "Accepted" };
        break;
      default:
        console.log(`[OCPP Client] Unknown action ${action}, sending empty response`);
    }

    // Send CallResult
    const responseMessage = [3, messageId, response];
    this.send(responseMessage);
  }

  private send(message: any): void {
    if (this.ws && this.connected) {
      const data = JSON.stringify(message);
      console.log(`[OCPP Client] Sending:`, data);
      this.ws.send(data);
    }
  }

  private async call(action: string, payload: any): Promise<any> {
    return new Promise((resolve, reject) => {
      const messageId = (++this.messageId).toString();
      const message = [2, messageId, action, payload];

      // Setup callback for response
      this.pendingCallbacks.set(messageId, resolve);

      // Send message
      this.send(message);

      // Timeout after 10 seconds
      setTimeout(() => {
        if (this.pendingCallbacks.has(messageId)) {
          this.pendingCallbacks.delete(messageId);
          reject(new Error(`Timeout waiting for response to ${action}`));
        }
      }, 10000);
    });
  }

  async bootNotification(model = "Simulator", vendor = "evcc-test"): Promise<any> {
    return this.call("BootNotification", {
      chargePointModel: model,
      chargePointVendor: vendor,
      chargePointSerialNumber: this.stationId,
      firmwareVersion: "1.0.0",
    });
  }

  async statusNotification(
    connectorId: number,
    status: string,
    errorCode = "NoError"
  ): Promise<any> {
    return this.call("StatusNotification", {
      connectorId,
      status,
      errorCode,
      timestamp: new Date().toISOString(),
    });
  }

  async meterValues(connectorId: number, power: number, energy: number): Promise<any> {
    return this.call("MeterValues", {
      connectorId,
      meterValue: [
        {
          timestamp: new Date().toISOString(),
          sampledValue: [
            {
              value: power.toString(),
              context: "Sample.Periodic",
              format: "Raw",
              measurand: "Power.Active.Import",
              phase: null,
              location: "Outlet",
              unit: "W",
            },
            {
              value: energy.toString(),
              context: "Sample.Periodic",
              format: "Raw",
              measurand: "Energy.Active.Import.Register",
              phase: null,
              location: "Outlet",
              unit: "Wh",
            },
          ],
        },
      ],
    });
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
      this.connected = false;
    }
  }

  isConnected(): boolean {
    return this.connected;
  }

  getStationId(): string {
    return this.stationId;
  }

  getServerUrl(): string {
    return this.serverUrl;
  }
}
