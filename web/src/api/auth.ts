/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { getApiUrl } from '@/utils/baseUrl';

interface User {
  id: number;
  username: string;
}

interface LoginResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  user: User;
}

interface UserResponse {
  user: User | null;
}

interface RegistrationStatus {
  registrationEnabled: boolean;
  hasUsers: boolean;
  oidcEnabled: boolean;
}

export async function login(username: string, password: string): Promise<LoginResponse> {
  try {
    const response = await fetch(getApiUrl('/auth/login'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(error);
    }

    const data: LoginResponse = await response.json();
    return data;
  } catch (error) {
    console.error('Login error:', error);
    throw error;
  }
}

export async function register(username: string, password: string): Promise<LoginResponse> {
  try {
    const response = await fetch(getApiUrl('/auth/register'), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(error);
    }

    const data: LoginResponse = await response.json();
    return data;
  } catch (error) {
    console.error('Register error:', error);
    throw error;
  }
}

export async function logout(): Promise<void> {
  try {
    const response = await fetch(getApiUrl('/auth/logout'), {
      method: 'POST',
      credentials: 'include',
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(error);
    }
  } catch (error) {
    console.error('Logout error:', error);
    throw error;
  }
}

export async function verify(): Promise<boolean> {
  try {
    const response = await fetch(getApiUrl('/auth/verify'), {
      credentials: 'include',
    });

    if (!response.ok) {
      if (response.status === 401) {
        return false;
      }
      throw new Error('Session verification failed');
    }

    return true;
  } catch (error) {
    console.error('Session verification error:', error);
    return false;
  }
}

export async function getUserInfo(): Promise<UserResponse> {
  try {
    const response = await fetch(getApiUrl('/auth/user'), {
      credentials: 'include',
    });

    if (!response.ok) {
      return { user: null };
    }

    const data: UserResponse = await response.json();
    return data;
  } catch (error) {
    console.error('User info retrieval error:', error);
    return { user: null };
  }
}

export async function checkRegistrationStatus(): Promise<RegistrationStatus> {
  try {
    const response = await fetch(getApiUrl('/auth/status'), {
      credentials: 'include',
    });
    
    if (response.ok) {
      const data = await response.json();
      return {
        registrationEnabled: data.registrationEnabled ?? false,
        hasUsers: data.hasUsers ?? true,
        oidcEnabled: data.oidcEnabled ?? false
      };
    }

    // If the endpoint fails, assume users exist to prevent unwanted registration
    return {
      registrationEnabled: false,
      hasUsers: true,
      oidcEnabled: false
    };
  } catch (error) {
    console.error('Registration status check failed:', error);
    // Default to secure state
    return {
      registrationEnabled: false,
      hasUsers: true,
      oidcEnabled: false
    };
  }
}

export function getOIDCLoginUrl(): string {
  return getApiUrl('/auth/oidc/login');
}
