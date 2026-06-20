import React, { useEffect } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { useAuthStore } from '../store/authStore';
import { useSyncStore } from '../store/syncStore';
import { useOfflineSync } from '../hooks/useOfflineSync';

import LoginScreen from '../screens/LoginScreen';
import DashboardScreen from '../screens/DashboardScreen';
import WorkOrderDetailScreen from '../screens/WorkOrderDetailScreen';
import ChecklistScreen from '../screens/ChecklistScreen';
import PhotoCaptureScreen from '../screens/PhotoCaptureScreen';
import SignatureScreen from '../screens/SignatureScreen';
import ProfileScreen from '../screens/ProfileScreen';
import QRScannerScreen from '../screens/QRScannerScreen';

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
          }

          return <Ionicons name={iconName} size={size} color={color} />;
        },
        tabBarActiveTintColor: '#2563eb',
        tabBarInactiveTintColor: 'gray',
        headerStyle: {
          backgroundColor: '#1e40af',
        },
        headerTintColor: '#fff',
      })}
    >
      <Tab.Screen
        name="Dashboard"
        component={DashboardScreen}
        options={{ title: 'Мои задания' }}
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
  useOfflineSync();

  useEffect(() => {
    loadStoredAuth();
    loadQueue();
  }, []);

  if (isLoading) {
    return null;
  }

  return (
    <NavigationContainer>
      <Stack.Navigator
        screenOptions={{
          headerStyle: { backgroundColor: '#1e40af' },
          headerTintColor: '#fff',
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
              options={{ title: 'Наряд-заказ' }}
            />
            <Stack.Screen
              name="Checklist"
              component={ChecklistScreen}
              options={{ title: 'Чек-лист' }}
            />
            <Stack.Screen
              name="PhotoCapture"
              component={PhotoCaptureScreen}
              options={{ title: 'Фотофиксация' }}
            />
            <Stack.Screen
              name="Signature"
              component={SignatureScreen}
              options={{ title: 'Подпись клиента' }}
            />
            <Stack.Screen
              name="QRScanner"
              component={QRScannerScreen}
              options={{ title: 'Сканер QR' }}
            />
          </>
        )}
      </Stack.Navigator>
    </NavigationContainer>
  );
}