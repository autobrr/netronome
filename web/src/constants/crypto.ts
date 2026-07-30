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
  /** Who receives it. Shown where addresses from both maintainers are mixed. */
  owner: string;
}

export const SOUP_CRYPTO: CryptoAddress[] = [
  {
    name: "Bitcoin",
    symbol: "BTC",
    address: "bc1qduy50dhnaw8eexkr34f9z8eczlezkalcavrd56",
    verifiable: true,
    owner: "s0up",
  },
  {
    name: "Ethereum",
    symbol: "ETH",
    address: "0x570179b3331777D027052Bb37a8E8D9A085c52Dc",
    verifiable: true,
    owner: "s0up",
  },
  {
    name: "Litecoin",
    symbol: "LTC",
    address: "ltc1qh7yv6pkkf8hlrauxhpnh02z2nnhspw05smlwvn",
    verifiable: true,
    owner: "s0up",
  },
  {
    name: "Monero",
    symbol: "XMR",
    address:
      "8AMPTPgjmLG9armLBvRA8NMZqPWuNT4US3kQoZrxDDVSU21kpYpFr1UCWmmtcBKGsvDCFA3KTphGXExWb3aHEu67JkcjAvC",
    verifiable: false,
    owner: "s0up",
  },
];

/**
 * XMR carries over from before netronome had its own wallets - it is settled by
 * hand, so there is no verifier allowlist for it to collide with.
 *
 * BTC/ETH/LTC are pending new wallets. They have to be new: productForAddress
 * checks netronome's list first, so reusing his qui addresses would route qui's
 * own donors to netronome. Add them here with verifiable: true and they show up
 * in both dialogs on their own.
 */
export const ZZE0S_CRYPTO: CryptoAddress[] = [
  {
    name: "Monero",
    symbol: "XMR",
    address:
      "44AvbWXzFN3bnv2oj92AmEaR26PQf5Ys4W155zw3frvEJf2s4g325bk4tRBgH7umSVMhk88vkU3gw9cDvuCSHgpRPsuWVJp",
    verifiable: false,
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
  .filter((c) => c.verifiable)
  .sort(
    (a, b) => SYMBOL_ORDER.indexOf(a.symbol) - SYMBOL_ORDER.indexOf(b.symbol)
  );
