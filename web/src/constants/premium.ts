/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/**
 * Polar checkout for the premium theme license.
 *
 * A public buy link, not a credential - it is meant to be shared. The Polar
 * organization id is the build-time secret, injected via -X main.PolarOrgID.
 */
export const POLAR_CHECKOUT_URL =
  "https://buy.polar.sh/polar_cl_laGHY1sxRB9p9s4bnPz1ZUxaC5Uk8dm6dAB66116IcK";

/**
 * Polar's customer portal for this organization.
 *
 * The way out of "activation limit already reached": activations are seats,
 * and a reinstall onto a fresh config directory burns one. Only the customer
 * can release an old activation, and only from here.
 */
export const POLAR_PORTAL_URL = "https://polar.sh/netronome/portal";

/**
 * Where a crypto donation is turned into a license.
 *
 * The buyer proves the transaction there and gets back a checkout link with a
 * 100% discount code already in the query string, so the key still comes from
 * the normal Polar checkout and activation is unchanged.
 */
export const CRYPTO_VERIFIER_URL = "https://crypto.netrono.me";

export const PREMIUM_MIN_PRICE = "$4.99";
