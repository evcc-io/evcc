declare global {
    interface State {
        offline: boolean;
        loadpoints: [];
        forecast?: any
        currency?: CURRENCY
    }

    interface Window {
        app: any
    }
}

export enum CURRENCY {
    EUR = "EUR",
    USD = "USD",
}
