// Server manager service
// Manages connections to multiple Sovereign servers

export interface ServerConnection {
  id: string;
  url: string;
  name: string;
  connected: boolean;
}

// TODO: Multi-server connection management
// TODO: Unified message stream
export {};
