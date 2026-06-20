import React, { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  Image,
  FlatList,
  Alert,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { NativeStackScreenProps, NativeStackNavigationProp } from '@react-navigation/native-stack';
import * as ImagePicker from 'expo-image-picker';
import { RootStackParamList } from '../types';
import { useUploadPhoto } from '../hooks/useWorkOrders';
import { useLocation } from '../hooks/useLocation';

type Props = NativeStackScreenProps<RootStackParamList, 'PhotoCapture'>;

export default function PhotoCaptureScreen({ route }: Props) {
  const { workOrder, checklist } = route.params;
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const uploadMutation = useUploadPhoto();
  const { latitude, longitude, loading: locationLoading } = useLocation();

  const [photos, setPhotos] = useState<string[]>([]);
  const [uploading, setUploading] = useState(false);

  const takePhoto = async () => {
    const permission = await ImagePicker.requestCameraPermissionsAsync();
    if (!permission.granted) {
      Alert.alert('Ошибка', 'Необходимо разрешение на использование камеры');
      return;
    }

    const result = await ImagePicker.launchCameraAsync({
      quality: 0.8,
      allowsEditing: false,
    });

    if (!result.canceled && result.assets.length > 0) {
      setPhotos((prev) => [...prev, result.assets[0].uri]);
    }
  };

  const pickFromGallery = async () => {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      Alert.alert('Ошибка', 'Необходимо разрешение на доступ к галерее');
      return;
    }

    const result = await ImagePicker.launchImageLibraryAsync({
      quality: 0.8,
      allowsEditing: false,
    });

    if (!result.canceled && result.assets.length > 0) {
      setPhotos((prev) => [...prev, result.assets[0].uri]);
    }
  };

  const removePhoto = (index: number) => {
    setPhotos((prev) => prev.filter((_, i) => i !== index));
  };

  const handleContinue = async () => {
    if (photos.length === 0) {
      Alert.alert('Предупреждение', 'Добавьте хотя бы одно фото');
      return;
    }

    setUploading(true);
    const uploadedUrls: string[] = [];

    try {
      for (const photoUri of photos) {
        const result = await uploadMutation.mutateAsync({
          workOrderId: workOrder.id,
          photoUri,
        });
        uploadedUrls.push(result.url);
      }
    } catch (error) {
      Alert.alert('Ошибка', 'Не удалось загрузить фото. Используются локальные URI.');
      uploadedUrls.push(...photos);
    }

    setUploading(false);

    navigation.navigate('Verification', {
      workOrder,
      checklist,
      photos: uploadedUrls,
    });
  };

  return (
    <View style={styles.container}>
      <View style={styles.locationBar}>
        <Text style={styles.locationText}>
          {locationLoading
            ? 'Определение координат...'
            : `📍 ${latitude.toFixed(5)}, ${longitude.toFixed(5)}`}
        </Text>
      </View>

      <FlatList
        data={photos}
        keyExtractor={(_, index) => index.toString()}
        horizontal
        contentContainerStyle={styles.photoList}
        renderItem={({ item, index }) => (
          <View style={styles.photoWrapper}>
            <Image source={{ uri: item }} style={styles.photo} />
            <TouchableOpacity
              style={styles.removeButton}
              onPress={() => removePhoto(index)}
            >
              <Text style={styles.removeText}>✕</Text>
            </TouchableOpacity>
          </View>
        )}
        ListEmptyComponent={
          <View style={styles.emptyPhotos}>
            <Text style={styles.emptyText}>Нет фото. Добавьте снимки ↓</Text>
          </View>
        }
      />

      <View style={styles.captureRow}>
        <TouchableOpacity style={styles.captureButton} onPress={takePhoto}>
          <Text style={styles.captureButtonText}>📷 Камера</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.captureButton} onPress={pickFromGallery}>
          <Text style={styles.captureButtonText}>🖼 Галерея</Text>
        </TouchableOpacity>
      </View>

      <TouchableOpacity
        style={[styles.continueButton, uploading && styles.buttonDisabled]}
        onPress={handleContinue}
        disabled={uploading}
        activeOpacity={0.7}
      >
        {uploading ? (
          <ActivityIndicator color="#fff" />
        ) : (
          <Text style={styles.continueButtonText}>
            Продолжить → Подпись ({photos.length} фото)
          </Text>
        )}
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f1f5f9',
  },
  locationBar: {
    backgroundColor: '#fff',
    padding: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#e2e8f0',
  },
  locationText: {
    fontSize: 12,
    color: '#64748b',
  },
  photoList: {
    padding: 16,
    gap: 12,
  },
  photoWrapper: {
    position: 'relative',
  },
  photo: {
    width: 200,
    height: 200,
    borderRadius: 12,
    backgroundColor: '#e2e8f0',
  },
  removeButton: {
    position: 'absolute',
    top: -6,
    right: -6,
    backgroundColor: '#dc2626',
    width: 24,
    height: 24,
    borderRadius: 12,
    justifyContent: 'center',
    alignItems: 'center',
  },
  removeText: {
    color: '#fff',
    fontSize: 12,
    fontWeight: 'bold',
  },
  emptyPhotos: {
    width: 200,
    height: 200,
    borderRadius: 12,
    borderWidth: 2,
    borderColor: '#e2e8f0',
    borderStyle: 'dashed',
    justifyContent: 'center',
    alignItems: 'center',
  },
  emptyText: {
    fontSize: 14,
    color: '#94a3b8',
    textAlign: 'center',
  },
  captureRow: {
    flexDirection: 'row',
    padding: 16,
    gap: 12,
  },
  captureButton: {
    flex: 1,
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#e2e8f0',
  },
  captureButtonText: {
    fontSize: 16,
    fontWeight: '500',
    color: '#1e293b',
  },
  continueButton: {
    backgroundColor: '#2563eb',
    margin: 16,
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  continueButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});