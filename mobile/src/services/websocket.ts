// WebSocket client service
// Manages connection to a single Sovereign server

export interface WebSocketConfig {
  url: string;
  token: string;
  onMessage: (data: Uint8Array) => void;
  onDisconnect: () => void;
}

// TODO: Implement WebSocket client with reconnection logic
export {};
