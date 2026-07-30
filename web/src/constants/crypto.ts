/*
 * Copyright (c) 2024-2026, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/**
 * Donation addresses, shared by the donate dialog and the premium purchase flow.
 *
 * These are netronome's own wallets, deliberately separate from qui's: the crypto
 * verifier decides which product a payment bought from the address the chain says
 * was paid, so the two sets must never overlap.
 *
 * Everything except XMR can be checked on-chain at crypto.netrono.me and turned
 * into a discount code automatically. XMR cannot - Monero hides amounts and
 * recipients from outside observers, so it is settled by hand.
 */
export interface CryptoAddress {
  name: string;
  symbol: "BTC" | "ETH" | "LTC" | "XMR";
  address: string;
  /** Who receives it. Shown where addresses from both maintainers are mixed. */
  owner: string;
}

export const SOUP_CRYPTO: CryptoAddress[] = [
  {
    name: "Bitcoin",
    symbol: "BTC",
    address: "bc1qduy50dhnaw8eexkr34f9z8eczlezkalcavrd56",
    owner: "s0up",
  },
  {
    name: "Ethereum",
    symbol: "ETH",
    address: "0x570179b3331777D027052Bb37a8E8D9A085c52Dc",
    owner: "s0up",
  },
  {
    name: "Litecoin",
    symbol: "LTC",
    address: "ltc1qh7yv6pkkf8hlrauxhpnh02z2nnhspw05smlwvn",
    owner: "s0up",
  },
  {
    name: "Monero",
    symbol: "XMR",
    address:
      "8AMPTPgjmLG9armLBvRA8NMZqPWuNT4US3kQoZrxDDVSU21kpYpFr1UCWmmtcBKGsvDCFA3KTphGXExWb3aHEu67JkcjAvC",
    owner: "s0up",
  },
];

/**
 * BTC/ETH/LTC are netronome-only wallets, deliberately not his qui ones:
 * productForAddress checks netronome's list first, so a shared address would
 * route qui's own donors to netronome. XMR carries over from before netronome
 * had its own wallets - settled by hand, no allowlist to collide with.
 */
export const ZZE0S_CRYPTO: CryptoAddress[] = [
  {
    name: "Bitcoin",
    symbol: "BTC",
    address: "bc1qx7usmx4v2ek6wd6azqrhw8t88eqc7jnmjfwfkg",
    owner: "zze0s",
  },
  {
    name: "Ethereum",
    symbol: "ETH",
    address: "0xc1C761c00dec2Bb2D76A0a05df2fde16aD1A8A75",
    owner: "zze0s",
  },
  {
    name: "Litecoin",
    symbol: "LTC",
    address: "ltc1qyf0kfrt76j6y254gyjkt6tchg4sqngwtsa5q9t",
    owner: "zze0s",
  },
  {
    name: "Monero",
    symbol: "XMR",
    address:
      "44AvbWXzFN3bnv2oj92AmEaR26PQf5Ys4W155zw3frvEJf2s4g325bk4tRBgH7umSVMhk88vkU3gw9cDvuCSHgpRPsuWVJp",
    owner: "zze0s",
  },
];

const SYMBOL_ORDER: CryptoAddress["symbol"][] = ["BTC", "ETH", "LTC", "XMR"];

/**
 * What the verifier can settle automatically, across every maintainer. Sorted
 * by coin rather than by owner so the purchase flow reads as "pick a coin" -
 * both addresses for one coin sit together, and sort is stable, so s0up stays
 * first within each pair.
 */
export const VERIFIABLE_CRYPTO = [...SOUP_CRYPTO, ...ZZE0S_CRYPTO]
  .filter((c) => c.symbol !== "XMR")
  .sort(
    (a, b) => SYMBOL_ORDER.indexOf(a.symbol) - SYMBOL_ORDER.indexOf(b.symbol)
  );
