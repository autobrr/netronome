/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

export function getBaseUrl(): string {
    // Get from window.__BASE_URL__ which will be set in index.html
    const baseUrl = window.__BASE_URL__;
    // Return '/' for empty or root paths
    if (!baseUrl || baseUrl === '/') return '';
    // Otherwise ensure no trailing slash
    return baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
}

export function getApiUrl(path: string): string {
    const baseUrl = getBaseUrl();
    const apiPath = path.startsWith('/') ? path : `/${path}`;
    // When base URL is empty, don't add an extra slash
    return baseUrl ? `${baseUrl}/api${apiPath}` : `/api${apiPath}`;
} 