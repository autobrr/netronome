export function getBaseUrl(): string {
    // Get from window.__BASE_URL__ which will be set in index.html
    const baseUrl = window.__BASE_URL__ || '';
    return baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
}

export function getApiUrl(path: string): string {
    const baseUrl = getBaseUrl();
    const apiPath = path.startsWith('/') ? path : `/${path}`;
    return `${baseUrl}/api${apiPath}`;
} 