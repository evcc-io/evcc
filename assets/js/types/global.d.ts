declare global {
    interface State {
        offline: boolean;
        loadpoints: [];
    }

    interface Window {
        app: any
    }
}

export { };