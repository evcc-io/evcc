import { baseApi } from "@/api";

// input the provider needs during a server-side login, e.g. a captcha
export interface AuthChallenge {
  kind: "captcha" | "code";
  image?: string;
  link?: string;
}

export type AuthState = {
  ok: boolean;
  loading: boolean;
  error: string | null;
  providerId: string | null;
  providerUrl: string | null;
  code: string | null;
  expiry: Date | null;
  challenge: AuthChallenge | null;
};

// one of three shapes: redirect url, device code, or challenge
export type ProviderLoginResponse = {
  loginUri?: string;
  code?: string;
  expiry?: string;
  challenge?: AuthChallenge;
  authenticated?: boolean;
  error?: string;
};

export const initialAuthState = (): AuthState => ({
  ok: false,
  loading: false,
  error: null,
  providerId: null,
  providerUrl: null,
  code: null,
  expiry: null,
  challenge: null,
});

// run a login step and map the response onto the state
const loginRequest = async (
  state: AuthState,
  request: () => Promise<{ status: number; data: ProviderLoginResponse }>
) => {
  try {
    state.loading = true;
    state.error = null;

    const { status, data } = await request();
    if (status !== 200) {
      state.error = data?.error ?? "Login failed";
      return { success: false, error: state.error };
    }

    state.providerUrl = data.loginUri || null;
    state.code = data.code || null;
    state.expiry = data.expiry ? new Date(data.expiry) : null;
    state.challenge = data.challenge || null;
    if (data.authenticated) {
      state.ok = true;
    }
    return { success: true, data };
  } catch (e: any) {
    console.error("login request failed", e);
    state.error = e.message || "Unexpected login error";
    return { success: false, error: state.error };
  } finally {
    state.loading = false;
  }
};

export const prepareAuthLogin = (state: AuthState, providerId: string) => {
  state.providerId = providerId;

  let url = `providerauth/login?id=${encodeURIComponent(providerId)}`;
  // restore the config modal stack on callback
  const returnTo = window.location.hash.split("?")[1];
  if (returnTo) {
    url += `&return=${encodeURIComponent(returnTo)}`;
  }

  return loginRequest(state, () =>
    baseApi.get<ProviderLoginResponse>(url, {
      validateStatus: (code) => [200, 400].includes(code),
    })
  );
};

// answer the current challenge; the response is the next step or completion
export const submitAuthChallenge = (state: AuthState, answer: string) =>
  loginRequest(state, () =>
    baseApi.post<ProviderLoginResponse>(
      `providerauth/submit?id=${encodeURIComponent(state.providerId ?? "")}`,
      { answer },
      { validateStatus: (code) => [200, 400].includes(code) }
    )
  );

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
