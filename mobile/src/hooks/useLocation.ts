import { useState, useEffect } from 'react';
import * as Location from 'expo-location';

interface LocationState {
  latitude: number;
  longitude: number;
  error: string | null;
  loading: boolean;
}

export function useLocation() {
  const [location, setLocation] = useState<LocationState>({
    latitude: 0,
    longitude: 0,
    error: null,
    loading: true,
  });

  useEffect(() => {
    let mounted = true;

    (async () => {
      const { status } = await Location.requestForegroundPermissionsAsync();
      if (status !== 'granted') {
        if (mounted) {
          setLocation((prev) => ({
            ...prev,
            error: 'Permission denied',
            loading: false,
          }));
        }
        return;
      }

      try {
        const loc = await Location.getCurrentPositionAsync({
          accuracy: Location.Accuracy.Balanced,
        });
        if (mounted) {
          setLocation({
            latitude: loc.coords.latitude,
            longitude: loc.coords.longitude,
            error: null,
            loading: false,
          });
        }
      } catch (err) {
        if (mounted) {
          setLocation((prev) => ({
            ...prev,
            error: 'Failed to get location',
            loading: false,
          }));
        }
      }
    })();

    return () => {
      mounted = false;
    };
  }, []);

  const refreshLocation = async () => {
    setLocation((prev) => ({ ...prev, loading: true }));
    try {
      const loc = await Location.getCurrentPositionAsync({
        accuracy: Location.Accuracy.Balanced,
      });
      setLocation({
        latitude: loc.coords.latitude,
        longitude: loc.coords.longitude,
        error: null,
        loading: false,
      });
    } catch {
      setLocation((prev) => ({
        ...prev,
        error: 'Failed to refresh location',
        loading: false,
      }));
    }
  };

  return { ...location, refreshLocation };
}