import { baseApi } from "@/api";

export type AuthState = {
  ok: boolean;
  loading: boolean;
  error: string | null;
  providerUrl: string | null;
  providerId: string | null;
  code: string | null;
  codeInput: { message?: string; sent?: boolean } | null;
  expiry: Date | null;
};

export type ProviderLoginResponse = {
  loginUri?: string;
  code?: string;
  codeInput?: { message?: string; sent?: boolean };
  expiry?: string;
  error?: string;
};

export type AuthProviderMessages = {
  loginFailed: string;
  unexpectedLoginError: string;
  missingProvider: string;
  codeVerificationFailed: string;
  unexpectedVerificationError: string;
  logoutFailed: string;
  unexpectedLogoutError: string;
};

export const initialAuthState = (): AuthState => ({
  ok: false,
  loading: false,
  error: null,
  providerUrl: null,
  providerId: null,
  code: null,
  codeInput: null,
  expiry: null,
});

export const prepareAuthLogin = async (
  state: AuthState,
  providerId: string,
  messages: AuthProviderMessages,
  sendCode = false
) => {
  try {
    state.loading = true;
    state.error = null;

    const params = new URLSearchParams({ id: providerId });
    if (sendCode) {
      params.set("sendCode", "true");
    }
    const url = `providerauth/login?${params.toString()}`;
    const { status, data } = await baseApi.get<ProviderLoginResponse>(url, {
      validateStatus: (code) => [200, 400].includes(code),
    });

    if (status === 200) {
      state.providerId = providerId;
      state.providerUrl = data.loginUri || null;
      state.code = data.code || null;
      state.codeInput = data.codeInput || null;
      state.expiry = data.expiry ? new Date(data.expiry) : null;
      state.ok = !state.providerUrl && !state.code && !state.codeInput;
      return { success: true, data };
    } else {
      state.error = data?.error ?? messages.loginFailed;
      return { success: false, error: state.error };
    }
  } catch (e: any) {
    console.error("prepareAuthLogin failed", e);
    state.error = e.message || messages.unexpectedLoginError;
    return { success: false, error: state.error };
  } finally {
    state.loading = false;
  }
};

export const requestAuthCode = async (state: AuthState, messages: AuthProviderMessages) => {
  if (!state.providerId) {
    state.error = messages.missingProvider;
    return { success: false, error: state.error };
  }

  return prepareAuthLogin(state, state.providerId, messages, true);
};

export const submitAuthCode = async (
  state: AuthState,
  code: string,
  messages: AuthProviderMessages
) => {
  if (!state.providerId) {
    state.error = messages.missingProvider;
    return { success: false, error: state.error };
  }

  try {
    state.loading = true;
    state.error = null;

    const url = `providerauth/code?id=${encodeURIComponent(state.providerId)}`;
    const { status, data } = await baseApi.post(
      url,
      { code },
      {
        validateStatus: (code) => [200, 400].includes(code),
      }
    );

    if (status === 200) {
      state.ok = true;
      state.codeInput = null;
      state.code = null;
      state.providerUrl = null;
      return { success: true };
    }

    state.error = data?.error ?? messages.codeVerificationFailed;
    return { success: false, error: state.error };
  } catch (e: any) {
    console.error("submitAuthCode failed", e);
    state.error = e.message || messages.unexpectedVerificationError;
    return { success: false, error: state.error };
  } finally {
    state.loading = false;
  }
};

export const performAuthLogout = async (providerId: string, messages: AuthProviderMessages) => {
  try {
    const url = `providerauth/logout?id=${encodeURIComponent(providerId)}`;
    const { status, data } = await baseApi.get(url, {
      validateStatus: (code) => [200, 400, 500].includes(code),
    });

    if (status === 200) {
      return { success: true };
    } else {
      return { success: false, error: data?.error || messages.logoutFailed };
    }
  } catch (e: any) {
    console.error("performAuthLogout failed", e);
    return { success: false, error: e.message || messages.unexpectedLogoutError };
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

export const prepareAuthRedirect = async (
  state: AuthState,
  authId: string,
  messages: AuthProviderMessages
) => {
  try {
    state.loading = true;
    state.error = null;

    const url = `providerauth/redirect?id=${encodeURIComponent(authId)}`;
    const { data } = await baseApi.get<ProviderLoginResponse>(url);

    state.providerId = authId;
    state.providerUrl = data.loginUri || null;
    state.code = data.code || null;
    state.codeInput = data.codeInput || null;
    state.expiry = data.expiry ? new Date(data.expiry) : null;
    state.ok = !state.providerUrl && !state.code && !state.codeInput;
    return { success: true, data };
  } catch (e: any) {
    console.error("prepareAuthRedirect failed", e);
    state.error = e.message || messages.unexpectedLoginError;
    return { success: false, error: state.error };
  } finally {
    state.loading = false;
  }
};
