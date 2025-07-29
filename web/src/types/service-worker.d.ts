/// <reference lib="webworker" />

declare interface ExtendableEvent extends Event {
  waitUntil(fn: Promise<any>): void;
}

declare interface Args {
  [key: string]: any;
}