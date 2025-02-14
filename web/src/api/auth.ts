/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface LoginCredentials {
  username: string;
  password: string;
}

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
  user: User;
}

interface RegistrationStatus {
  registrationEnabled: boolean;
  hasUsers: boolean;
  oidcEnabled: boolean;
}

export async function login(credentials: LoginCredentials): Promise<User> {
  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(credentials),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(error);
  }

  const data: LoginResponse = await response.json();
  return data.user;
}

export async function register(credentials: LoginCredentials): Promise<User> {
  const response = await fetch('/api/auth/register', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include',
    body: JSON.stringify(credentials),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(error);
  }

  const data: { message: string; user: User } = await response.json();
  return data.user;
}

export async function logout(): Promise<void> {
  const response = await fetch('/api/auth/logout', {
    method: 'POST',
    credentials: 'include',
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(error);
  }
}

export async function verifySession(): Promise<User | null> {
  try {
    const response = await fetch('/api/auth/verify', {
      credentials: 'include',
    });

    if (!response.ok) {
      if (response.status === 401) {
        return null;
      }
      throw new Error('Session verification failed');
    }

    const userResponse = await fetch('/api/auth/user', {
      credentials: 'include',
    });

    if (!userResponse.ok) {
      return null;
    }

    const data: UserResponse = await userResponse.json();
    return data.user;
  } catch (error) {
    console.error('Session verification error:', error);
    return null;
  }
}

export async function checkRegistrationStatus(): Promise<RegistrationStatus> {
  try {
    const response = await fetch('/api/auth/status', {
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
