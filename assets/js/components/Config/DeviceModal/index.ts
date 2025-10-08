import type { DeviceType, MeterTemplateUsage } from "@/types/evcc";
import { ConfigType } from "@/types/evcc";
import api from "@/api";

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

export type TemplateParamUsage = "vehicle" | "battery" | "grid" | "pv" | "charger" | "aux" | "ext";

export type TemplateParam = {
  Name: string;
  Required: boolean;
  Advanced: boolean;
  Deprecated: boolean;
  Default?: string | number | boolean;
  Choice?: string[];
  Usages?: TemplateParamUsage[];
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
  icon?: string;
  deviceProduct?: string;
  yaml?: string;
  template: string | null;
  deviceTitle?: string;
  deviceIcon?: string;
  usage?: MeterTemplateUsage;
  heating?: boolean;
  integrateddevice?: boolean;
  stationid?: string;
  [key: string]: any;
};

export type ApiData = {
  type?: ConfigType;
  icon?: string;
  usage?: MeterTemplateUsage;
  title?: string;
  identifiers?: string[];
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

export function applyDefaultsFromTemplate(template: Template | null, values: DeviceValues) {
  const params = template?.Params || [];
  params
    .filter((p) => p.Default && !values[p.Name])
    .forEach((p) => {
      values[p.Name] = p.Default;
    });
}

export function customChargerName(type: ConfigType, isHeating: boolean) {
  if (!type) {
    return "config.general.customOption";
  }
  const prefix = "config.charger.type.";
  const suffix = isHeating ? ".heating" : ".charging";
  if (type === ConfigType.Custom) {
    return `${prefix}custom${suffix}`;
  }
  return `${prefix}${type}`;
}

export function createDeviceUtils(deviceType: DeviceType) {
  function test(id: number | undefined, data: any) {
    let url = `config/test/${deviceType}`;
    if (id !== undefined) {
      url += `/merge/${id}`;
    }
    return api.post(url, data, { timeout });
  }

  function update(id: number, data: any, force = false) {
    const params = { force };
    return api.put(`config/devices/${deviceType}/${id}`, data, { params });
  }

  function remove(id: number) {
    return api.delete(`config/devices/${deviceType}/${id}`);
  }

  async function load(id: number) {
    const response = await api.get(`config/devices/${deviceType}/${id}`);
    return response.data;
  }

  async function create(data: any, force = false) {
    const params = { force };
    const response = await api.post(`config/devices/${deviceType}`, data, { params });
    return response.data;
  }

  async function loadProducts(lang?: string, usage?: string) {
    const params: Record<string, string | undefined> = { lang };
    if (usage) {
      params["usage"] = usage;
    }
    const response = await api.get(`config/products/${deviceType}`, { params });
    return response.data;
  }

  async function loadTemplate(templateName: string, lang?: string) {
    if (!templateName) return null;

    const opts = {
      params: {
        lang,
        name: templateName,
      },
    };
    const response = await api.get(`config/templates/${deviceType}`, opts);
    return response.data;
  }

  return {
    test,
    update,
    remove,
    load,
    create,
    loadProducts,
    loadTemplate,
  };
}
