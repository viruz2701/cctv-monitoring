import React, { useEffect } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { useAuthStore } from '../store/authStore';
import { useSyncStore } from '../store/syncStore';
import { useOfflineSync } from '../hooks/useOfflineSync';
import { useBackgroundSync } from '../hooks/useBackgroundSync';
import OfflineIndicator from '../components/OfflineIndicator';

import LoginScreen from '../screens/LoginScreen';
import DashboardScreen from '../screens/DashboardScreen';
import WorkOrderDetailScreen from '../screens/WorkOrderDetailScreen';
import ChecklistScreen from '../screens/ChecklistScreen';
import PhotoCaptureScreen from '../screens/PhotoCaptureScreen';
import SignatureScreen from '../screens/SignatureScreen';
import ProfileScreen from '../screens/ProfileScreen';
import QRScannerScreen from '../screens/QRScannerScreen';
import VerificationScreen from '../screens/VerificationScreen';

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
          } else if (route.name === 'Profile') {
            iconName = focused ? 'person' : 'person-outline';
          } else if (route.name === 'QRScanner') {
            iconName = focused ? 'qr-code' : 'qr-code-outline';
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
        // Gesture navigation: enable swipe back on all screens
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
        name="QRScanner"
        component={QRScannerScreen}
        options={{
          title: 'Сканер QR',
          tabBarIcon: ({ focused, color, size }: { focused: boolean; color: string; size: number }) => (
            <Ionicons name={focused ? 'qr-code' : 'qr-code-outline'} size={size} color={color} />
          ),
        }}
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
  }, []);

  if (isLoading) {
    return null;
  }

  return (
    <NavigationContainer>
      {/* Global offline indicator */}
      {!isOnline && <OfflineIndicator />}

      <Stack.Navigator
        screenOptions={{
          headerStyle: { backgroundColor: '#1e40af' },
          headerTintColor: '#fff',
          // Enable gesture navigation (swipe back) for all stack screens
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
                // Swipe down to go back
                gestureDirection: 'vertical',
                animation: 'slide_from_bottom',
              }}
            />
            <Stack.Screen
              name="Checklist"
              component={ChecklistScreen}
              options={{ title: 'Чек-лист' }}
            />
            <Stack.Screen
              name="PhotoCapture"
              component={PhotoCaptureScreen}
              options={{
                title: 'Фотофиксация',
                gestureDirection: 'vertical',
                animation: 'slide_from_bottom',
              }}
            />
            <Stack.Screen
              name="Verification"
              component={VerificationScreen}
              options={{ title: 'Верификация' }}
            />
            <Stack.Screen
              name="Signature"
              component={SignatureScreen}
              options={{
                title: 'Подпись клиента',
                gestureDirection: 'vertical',
                animation: 'slide_from_bottom',
              }}
            />
            <Stack.Screen
              name="QRScanner"
              component={QRScannerScreen}
              options={{
                title: 'Сканер QR',
                animation: 'fade',
              }}
            />
          </>
        )}
      </Stack.Navigator>
    </NavigationContainer>
  );
}
