export interface Server {
  id: string;
  name: string;
  host: string;
  distance: number;
  country: string;
}

export interface Schedule {
  id?: number;
  serverIds: string[];
  interval: string;
  lastRun?: string;
  nextRun: string;
  enabled: boolean;
  options: TestOptions;
  createdAt?: string;
}

export interface TestOptions {
  enableDownload: boolean;
  enableUpload: boolean;
  enablePacketLoss: boolean;
  serverIds: string[];
} 