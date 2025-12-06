import { baseApi } from "@/api";

export type AuthState = {
  ok: boolean;
  loading: boolean;
  error: string | null;
  providerUrl: string | null;
  code: string | null;
  expiry: Date | null;
};

export type ProviderLoginResponse = {
  loginUri?: string;
  code?: string;
  expiry?: string;
  error?: string;
};

export const initialAuthState = (): AuthState => ({
  ok: false,
  loading: false,
  error: null,
  providerUrl: null,
  code: null,
  expiry: null,
});

export const prepareAuthLogin = async (state: AuthState, providerId: string) => {
  try {
    state.loading = true;
    state.error = null;

    const url = `providerauth/login?id=${encodeURIComponent(providerId)}`;
    const { status, data } = await baseApi.get<ProviderLoginResponse>(url, {
      validateStatus: (code) => [200, 400].includes(code),
    });

    if (status === 200) {
      state.providerUrl = data.loginUri || null;
      state.code = data.code || null;
      state.expiry = data.expiry ? new Date(data.expiry) : null;
      return { success: true, data };
    } else {
      state.error = data?.error ?? "Login failed";
      return { success: false, error: state.error };
    }
  } catch (e: any) {
    console.error("prepareAuthLogin failed", e);
    state.error = e.message || "Unexpected login error";
    return { success: false, error: state.error };
  } finally {
    state.loading = false;
  }
};

export const performAuthLogout = async (providerId: string) => {
  try {
    const url = `providerauth/logout?id=${encodeURIComponent(providerId)}`;
    const { status, data } = await baseApi.get(url, {
      validateStatus: (code) => [200, 400, 500].includes(code),
    });

    if (status === 200) {
      return { success: true };
    } else {
      return { success: false, error: data?.error || "Logout failed" };
    }
  } catch (e: any) {
    console.error("performAuthLogout failed", e);
    return { success: false, error: e.message || "Unexpected logout error" };
  }
};

// Device authentication utilities (used in DeviceModalBase)
export type DeviceAuthResponse = {
  success: boolean;
  authId?: string;
  loginUri?: string;
  code?: string;
  expiry?: string;
  error?: string;
};

export const prepareAuthRedirect = async (state: AuthState, authId: string) => {
  try {
    state.loading = true;
    state.error = null;

    const url = `providerauth/redirect?id=${encodeURIComponent(authId)}`;
    const { data } = await baseApi.get<ProviderLoginResponse>(url);

    state.providerUrl = data.loginUri || null;
    state.code = data.code || null;
    state.expiry = data.expiry ? new Date(data.expiry) : null;
    return { success: true, data };
  } catch (e: any) {
    console.error("prepareAuthRedirect failed", e);
    state.error = e.message || "Unexpected login error";
    return { success: false, error: state.error };
  } finally {
    state.loading = false;
  }
};
