/// <reference lib="webworker" />

declare interface ExtendableEvent extends Event {
  waitUntil(fn: Promise<unknown>): void;
}

declare interface Args {
  [key: string]: unknown;
}