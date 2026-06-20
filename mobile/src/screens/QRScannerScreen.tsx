import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, Alert } from 'react-native';
import { BarCodeScanner } from 'expo-barcode-scanner';
import { useNavigation } from '@react-navigation/native';

export default function QRScannerScreen() {
  const navigation = useNavigation();
  const [hasPermission, setHasPermission] = useState<boolean | null>(null);
  const [scanned, setScanned] = useState(false);

  useEffect(() => {
    (async () => {
      const { status } = await BarCodeScanner.requestPermissionsAsync();
      setHasPermission(status === 'granted');
    })();
  }, []);

  const handleBarCodeScanned = ({ data }: { type: string; data: string }) => {
    if (scanned) return;
    setScanned(true);

    try {
      const parsed = JSON.parse(data);
      if (parsed.device_id) {
        Alert.alert('Устройство найдено', `ID: ${parsed.device_id}`, [
          { text: 'OK', onPress: () => navigation.goBack() },
        ]);
      } else {
        Alert.alert('QR-код', `Содержимое: ${data}`, [
          { text: 'Сканировать снова', onPress: () => setScanned(false) },
        ]);
      }
    } catch {
      Alert.alert('QR-код', `Содержимое: ${data}`, [
        { text: 'Сканировать снова', onPress: () => setScanned(false) },
      ]);
    }
  };

  if (hasPermission === null) {
    return (
      <View style={styles.centered}>
        <Text style={styles.infoText}>Запрос разрешения камеры...</Text>
      </View>
    );
  }

  if (hasPermission === false) {
    return (
      <View style={styles.centered}>
        <Text style={styles.errorText}>Нет доступа к камере</Text>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <BarCodeScanner
        onBarCodeScanned={scanned ? undefined : handleBarCodeScanned}
        style={StyleSheet.absoluteFillObject}
      />
      <View style={styles.overlay}>
        <View style={styles.scanFrame} />
        <Text style={styles.hint}>Наведите камеру на QR-код устройства</Text>
      </View>
      {scanned && (
        <View style={styles.rescanContainer}>
          <Text style={styles.rescanText} onPress={() => setScanned(false)}>
            Нажмите, чтобы сканировать снова
          </Text>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#000',
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#f1f5f9',
  },
  infoText: {
    fontSize: 16,
    color: '#64748b',
  },
  errorText: {
    fontSize: 16,
    color: '#dc2626',
  },
  overlay: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  scanFrame: {
    width: 250,
    height: 250,
    borderWidth: 2,
    borderColor: '#2563eb',
    borderRadius: 12,
    backgroundColor: 'transparent',
  },
  hint: {
    color: '#fff',
    fontSize: 14,
    marginTop: 20,
    backgroundColor: 'rgba(0,0,0,0.6)',
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 8,
    overflow: 'hidden',
  },
  rescanContainer: {
    position: 'absolute',
    bottom: 40,
    left: 0,
    right: 0,
    alignItems: 'center',
  },
  rescanText: {
    color: '#2563eb',
    fontSize: 16,
    fontWeight: '600',
    backgroundColor: 'rgba(255,255,255,0.9)',
    paddingHorizontal: 20,
    paddingVertical: 12,
    borderRadius: 8,
    overflow: 'hidden',
  },
});