import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import type { ConnectionState } from '../services/websocket';

interface ServerEntry {
  url: string;
  name: string;
  state: ConnectionState;
}

interface ServerListProps {
  serverUrl: string;
  connectionState: ConnectionState;
}

const ServerList: React.FC<ServerListProps> = ({ serverUrl, connectionState }) => {
  const servers: ServerEntry[] = [
    {
      url: serverUrl,
      name: 'Local Server',
      state: connectionState,
    },
  ];

  const statusColor = (state: ConnectionState): string => {
    switch (state) {
      case 'connected':
        return '#4CAF50';
      case 'connecting':
        return '#FF9800';
      case 'disconnected':
        return '#F44336';
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Servers</Text>
      {servers.map((server) => (
        <View key={server.url} style={styles.serverRow}>
          <View style={[styles.statusDot, { backgroundColor: statusColor(server.state) }]} />
          <View style={styles.serverInfo}>
            <Text style={styles.serverName}>{server.name}</Text>
            <Text style={styles.serverUrl}>{server.url}</Text>
          </View>
          <Text style={styles.stateLabel}>{server.state}</Text>
        </View>
      ))}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F5F5F5',
    padding: 16,
    paddingTop: 48,
  },
  title: {
    fontSize: 24,
    fontWeight: '700',
    color: '#1A1A2E',
    marginBottom: 16,
  },
  serverRow: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#FFFFFF',
    padding: 16,
    borderRadius: 12,
    marginBottom: 8,
    borderWidth: 1,
    borderColor: '#E0E0E0',
  },
  statusDot: {
    width: 12,
    height: 12,
    borderRadius: 6,
    marginRight: 12,
  },
  serverInfo: {
    flex: 1,
  },
  serverName: {
    fontSize: 16,
    fontWeight: '600',
    color: '#333333',
  },
  serverUrl: {
    fontSize: 12,
    color: '#999999',
    marginTop: 2,
  },
  stateLabel: {
    fontSize: 12,
    color: '#666666',
    textTransform: 'capitalize',
  },
});

export default ServerList;
