import type { DeviceType, MODBUS_COMSET, MeterTemplateUsage } from "@/types/evcc";
import { ConfigType } from "@/types/evcc";
import api from "@/api";
import { extractPlaceholders, replacePlaceholders } from "@/utils/placeholder";

export type Product = {
  group: string;
  name: string;
  template: string;
};

export type Template = {
  Params: TemplateParam[];
  Auth?: {
    type: string;
    params?: string[];
  };
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
  Service?: string;
  Usages?: TemplateParamUsage[];
};

export type ParamService = {
  name: string;
  dependencies: string[];
  url: (values: Record<string, any>) => string;
};

export type ModbusCapability = "rs485" | "tcpip";

export type ModbusParam = TemplateParam & {
  ID?: string;
  Comset?: MODBUS_COMSET;
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

export type AuthCheckResponse = {
  success: boolean;
  error?: string;
  authId?: string;
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

export async function loadServiceValues(path: string) {
  try {
    const response = await api.get(`/config/service/${path}`, {
      validateStatus: (status) => status >= 200 && status < 500,
    });
    return (response.data as string[]) || [];
  } catch {
    return [];
  }
}

export const createServiceEndpoints = (params: TemplateParam[]): ParamService[] => {
  return params
    .map((param) => {
      if (!param.Service) {
        return null;
      }
      const stringValues = (values: Record<string, any>): Record<string, string> =>
        Object.entries(values).reduce(
          (acc, [key, val]) => {
            if (val !== undefined && val !== null) acc[key] = String(val);
            return acc;
          },
          {} as Record<string, string>
        );

      return {
        name: param.Name,
        dependencies: extractPlaceholders(param.Service),
        url: (values: Record<string, any>) =>
          replacePlaceholders(param.Service!, stringValues(values)),
      } as ParamService;
    })
    .filter((endpoint): endpoint is ParamService => endpoint !== null);
};

export const fetchServiceValues = async (
  templateParams: TemplateParam[],
  values: DeviceValues
): Promise<Record<string, string[]>> => {
  const endpoints = createServiceEndpoints(templateParams);
  const result: Record<string, string[]> = {};

  await Promise.all(
    endpoints.map(async (endpoint) => {
      const params: Record<string, any> = {};
      endpoint.dependencies.forEach((dependency) => {
        if (values[dependency]) {
          params[dependency] = values[dependency];
        }
      });
      if (Object.keys(params).length !== endpoint.dependencies.length) {
        // missing dependency values, skip
        return;
      }
      const url = endpoint.url(params);
      const data = await loadServiceValues(url);
      if (data) {
        result[endpoint.name] = data;
      }
    })
  );

  return result;
};

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

  async function checkAuth(type: string, values: Record<string, any>): Promise<AuthCheckResponse> {
    const params = { type, ...values };
    try {
      const { status, data = {} } = await api.post(`config/auth`, params, {
        validateStatus: (status) => [204, 400].includes(status),
      });
      // already set up
      if (status === 204) {
        return { success: true };
      }
      // auth error, user has to perform login
      if (status === 400) {
        return { success: false, error: data?.error, authId: data?.loginRequired };
      }
    } catch (error) {
      return { success: false, error: (error as any).message };
    }
    return { success: false, error: "unexpected error" };
  }

  return {
    test,
    update,
    remove,
    load,
    create,
    loadProducts,
    loadTemplate,
    loadServiceValues,
    checkAuth,
  };
}
