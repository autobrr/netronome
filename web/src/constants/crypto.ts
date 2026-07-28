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
 * Anything marked verifiable can be checked on-chain at crypto.netrono.me and turned
 * into a discount code automatically. XMR cannot - it is settled by hand.
 */
export interface CryptoAddress {
  name: string;
  symbol: "BTC" | "ETH" | "LTC" | "XMR";
  address: string;
  verifiable: boolean;
}

export const SOUP_CRYPTO: CryptoAddress[] = [
  {
    name: "Bitcoin",
    symbol: "BTC",
    address: "bc1qduy50dhnaw8eexkr34f9z8eczlezkalcavrd56",
    verifiable: true,
  },
  {
    name: "Ethereum",
    symbol: "ETH",
    address: "0x570179b3331777D027052Bb37a8E8D9A085c52Dc",
    verifiable: true,
  },
  {
    name: "Litecoin",
    symbol: "LTC",
    address: "ltc1qh7yv6pkkf8hlrauxhpnh02z2nnhspw05smlwvn",
    verifiable: true,
  },
  {
    name: "Monero",
    symbol: "XMR",
    address:
      "8AMPTPgjmLG9armLBvRA8NMZqPWuNT4US3kQoZrxDDVSU21kpYpFr1UCWmmtcBKGsvDCFA3KTphGXExWb3aHEu67JkcjAvC",
    verifiable: false,
  },
];

/**
 * XMR is settled by hand, so this address carries over from before netronome
 * had its own wallets - there is no verifier allowlist for it to collide with.
 *
 * BTC/ETH/LTC are pending: they have to be new wallets, not the ones already
 * serving qui. productForAddress checks netronome's list first, so a shared
 * address would route qui's own donors to netronome.
 */
export const ZZE0S_CRYPTO: CryptoAddress[] = [
  {
    name: "Monero",
    symbol: "XMR",
    address:
      "44AvbWXzFN3bnv2oj92AmEaR26PQf5Ys4W155zw3frvEJf2s4g325bk4tRBgH7umSVMhk88vkU3gw9cDvuCSHgpRPsuWVJp",
    verifiable: false,
  },
];

/** The ones the verifier can settle automatically, across every maintainer. */
export const VERIFIABLE_CRYPTO = [...SOUP_CRYPTO, ...ZZE0S_CRYPTO].filter(
  (c) => c.verifiable
);
