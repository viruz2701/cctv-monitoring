import React, { useEffect, useRef } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { useAuthStore } from '../store/authStore';
import { useSyncStore } from '../store/syncStore';
import { useOfflineSync } from '../hooks/useOfflineSync';
import { useBackgroundSync } from '../hooks/useBackgroundSync';
import { syncService } from '../services/syncService';
import OfflineIndicator from '../components/OfflineIndicator';

import LoginScreen from '../screens/LoginScreen';
import DashboardScreen from '../screens/DashboardScreen';
import MapScreen from '../screens/MapScreen';
import WorkOrderDetailScreen from '../screens/WorkOrderDetailScreen';
import ProfileScreen from '../screens/ProfileScreen';
import QRScannerScreen from '../screens/QRScannerScreen';
import SignatureScreen from '../screens/SignatureScreen';
import CompleteWorkOrderWizard from '../components/CompleteWorkOrderWizard';

import { RootStackParamList, MainTabParamList } from '../types';

const Stack = createNativeStackNavigator<RootStackParamList>();
const Tab = createBottomTabNavigator<MainTabParamList>();

function MainTabs() {
  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        tabBarIcon: ({ focused, color, size }) => {
          let iconName: keyof typeof Ionicons.glyphMap = 'home';

          if (route.name === 'Dashboard') {
            iconName = focused ? 'list-circle' : 'list-circle-outline';
          } else if (route.name === 'Map') {
            iconName = focused ? 'map' : 'map-outline';
          } else if (route.name === 'Profile') {
            iconName = focused ? 'person' : 'person-outline';
          }

          return <Ionicons name={iconName} size={size} color={color} />;
        },
        tabBarActiveTintColor: '#2563eb',
        tabBarInactiveTintColor: 'gray',
        headerStyle: {
          backgroundColor: '#1e40af',
        },
        headerTintColor: '#fff',
        tabBarStyle: {
          borderTopLeftRadius: 16,
          borderTopRightRadius: 16,
          paddingBottom: 8,
          paddingTop: 4,
          height: 64,
        },
        gestureEnabled: true,
        gestureDirection: 'horizontal',
      })}
    >
      <Tab.Screen
        name="Dashboard"
        component={DashboardScreen}
        options={{ title: 'Мои задания' }}
      />
      <Tab.Screen
        name="Map"
        component={MapScreen}
        options={{ title: 'Карта объектов' }}
      />
      <Tab.Screen
        name="Profile"
        component={ProfileScreen}
        options={{ title: 'Профиль' }}
      />
    </Tab.Navigator>
  );
}

export default function AppNavigator() {
  const { isAuthenticated, isLoading, loadStoredAuth } = useAuthStore();
  const loadQueue = useSyncStore((s) => s.loadQueue);
  const isOnline = useSyncStore((s) => s.isOnline);

  // Offline sync on app state change
  useOfflineSync();

  // Background sync via Expo BackgroundFetch
  useBackgroundSync();

  useEffect(() => {
    loadStoredAuth();
    loadQueue();

    // Инициализация offline-first сервиса (SQLite + sync)
    syncService.initialize().catch((err) => {
      console.error('Failed to initialize sync service:', err);
    });

    return () => {
      syncService.destroy();
    };
  }, []);

  if (isLoading) {
    return null;
  }

  return (
    <NavigationContainer>
      {/* Global offline indicator — всегда активен, сам решает когда показываться */}
      <OfflineIndicator showQueueBadge />

      <Stack.Navigator
        screenOptions={{
          headerStyle: { backgroundColor: '#1e40af' },
          headerTintColor: '#fff',
          gestureEnabled: true,
          gestureDirection: 'horizontal',
          animation: 'slide_from_right',
        }}
      >
        {!isAuthenticated ? (
          <Stack.Screen
            name="Login"
            component={LoginScreen}
            options={{ headerShown: false }}
          />
        ) : (
          <>
            <Stack.Screen
              name="Main"
              component={MainTabs}
              options={{ headerShown: false }}
            />
            <Stack.Screen
              name="WorkOrderDetail"
              component={WorkOrderDetailScreen}
              options={{
                title: 'Наряд-заказ',
                gestureDirection: 'vertical',
                animation: 'slide_from_bottom',
              }}
            />
            <Stack.Screen
              name="CompleteWorkOrder"
              component={CompleteWorkOrderWizard}
              options={{
                title: 'Завершение наряда',
                // Disable back gesture during wizard — wizard handles navigation
                gestureEnabled: false,
                headerBackVisible: false,
                animation: 'slide_from_bottom',
              }}
            />
            <Stack.Screen
              name="QRScanner"
              component={QRScannerScreen}
              options={{
                title: 'Сканер QR',
                animation: 'fade',
                headerShown: false,
              }}
            />
            <Stack.Screen
              name="Signature"
              component={SignatureScreen}
              options={{
                title: 'Электронная подпись',
                animation: 'slide_from_bottom',
                headerShown: false,
              }}
            />
          </>
        )}
      </Stack.Navigator>
    </NavigationContainer>
  );
}
