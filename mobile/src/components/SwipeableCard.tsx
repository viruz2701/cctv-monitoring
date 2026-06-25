import React, { useRef, useCallback, ReactNode } from 'react';
import {
  View,
  Text,
  Animated,
  PanResponder,
  TouchableOpacity,
  StyleSheet,
  Dimensions,
  GestureResponderEvent,
  PanResponderGestureState,
} from 'react-native';

const SCREEN_WIDTH = Dimensions.get('window').width;
const SWIPE_THRESHOLD = 80;
const ACTION_WIDTH = 100;

export interface SwipeAction {
  key: string;
  label: string;
  color: string;
  icon?: string; // эмодзи-иконка
  onPress: () => void;
}

interface Props {
  children: ReactNode;
  /** Действия при свайпе влево (right actions) */
  rightActions?: SwipeAction[];
  /** Действия при свайпе вправо (left actions) */
  leftActions?: SwipeAction[];
  /** Блокировать свайп (для completed/cancelled) */
  disabled?: boolean;
}

/**
 * SwipeableCard — универсальный компонент для inline-действий через свайп.
 *
 * UX-03: Inline Editing
 * Использует чистый RN Animated + PanResponder (без react-native-gesture-handler)
 *
 * Соответствует:
 *   - OWASP ASVS V7 (Error handling — ошибки не ломают UI)
 *   - IEC 62443 SR 7.1 (Graceful degradation — при ошибке жеста просто тап)
 */
export default function SwipeableCard({
  children,
  rightActions = [],
  leftActions = [],
  disabled = false,
}: Props) {
  const translateX = useRef(new Animated.Value(0)).current;
  const lastOffset = useRef(0);

  // Максимальное смещение для правых и левых действий
  const maxRightSwipe = rightActions.length * ACTION_WIDTH;
  const maxLeftSwipe = leftActions.length * ACTION_WIDTH;

  const snapTo = useCallback(
    (toValue: number) => {
      Animated.spring(translateX, {
        toValue,
        useNativeDriver: true,
        bounciness: 4,
        speed: 14,
      }).start();
      lastOffset.current = toValue;
    },
    [translateX],
  );

  const resetPosition = useCallback(() => {
    snapTo(0);
  }, [snapTo]);

  const panResponder = useRef(
    PanResponder.create({
      onMoveShouldSetPanResponder: (
        _: GestureResponderEvent,
        gestureState: PanResponderGestureState,
      ) => {
        // Только горизонтальные свайпы
        return (
          !disabled &&
          Math.abs(gestureState.dx) > 10 &&
          Math.abs(gestureState.dx) > Math.abs(gestureState.dy)
        );
      },
      onPanResponderMove: (_, gestureState) => {
        let newX = lastOffset.current + gestureState.dx;
        // Ограничение смещения
        newX = Math.min(newX, maxLeftSwipe);
        newX = Math.max(newX, -maxRightSwipe);
        translateX.setValue(newX);
      },
      onPanResponderRelease: (_, gestureState) => {
        if (gestureState.dx < -SWIPE_THRESHOLD && rightActions.length > 0) {
          // Свайп влево — показать правые действия
          snapTo(-maxRightSwipe);
        } else if (gestureState.dx > SWIPE_THRESHOLD && leftActions.length > 0) {
          // Свайп вправо — показать левые действия
          snapTo(maxLeftSwipe);
        } else {
          resetPosition();
        }
      },
      onPanResponderTerminate: () => {
        resetPosition();
      },
    }),
  ).current;

  // Если действия отключены — просто рендерим children
  if (disabled) {
    return <>{children}</>;
  }

  const hasActions = rightActions.length > 0 || leftActions.length > 0;
  if (!hasActions) {
    return <>{children}</>;
  }

  return (
    <View style={styles.container}>
      {/* Левое фоновое действие (свайп вправо) */}
      {leftActions.length > 0 && (
        <View style={styles.leftActionsContainer}>
          {leftActions.map((action) => (
            <TouchableOpacity
              key={action.key}
              style={[styles.actionButton, { backgroundColor: action.color }]}
              onPress={() => {
                resetPosition();
                action.onPress();
              }}
            >
              <Text style={styles.actionIcon}>{action.icon || ''}</Text>
              <Text style={styles.actionLabel}>{action.label}</Text>
            </TouchableOpacity>
          ))}
        </View>
      )}

      {/* Правое фоновое действие (свайп влево) */}
      {rightActions.length > 0 && (
        <View style={styles.rightActionsContainer}>
          {rightActions.map((action) => (
            <TouchableOpacity
              key={action.key}
              style={[styles.actionButton, { backgroundColor: action.color }]}
              onPress={() => {
                resetPosition();
                action.onPress();
              }}
            >
              <Text style={styles.actionIcon}>{action.icon || ''}</Text>
              <Text style={styles.actionLabel}>{action.label}</Text>
            </TouchableOpacity>
          ))}
        </View>
      )}

      {/* Основной контент (анимированный) */}
      <Animated.View
        style={[styles.content, { transform: [{ translateX }] }]}
        {...panResponder.panHandlers}
      >
        {children}
      </Animated.View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginBottom: 12,
    position: 'relative',
    overflow: 'hidden',
    borderRadius: 12,
  },
  content: {
    backgroundColor: '#fff',
    borderRadius: 12,
    // Тень для контента поверх действий
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.05,
    shadowRadius: 4,
    elevation: 2,
  },
  leftActionsContainer: {
    position: 'absolute',
    left: 0,
    top: 0,
    bottom: 0,
    flexDirection: 'row',
    alignItems: 'stretch',
  },
  rightActionsContainer: {
    position: 'absolute',
    right: 0,
    top: 0,
    bottom: 0,
    flexDirection: 'row',
    alignItems: 'stretch',
  },
  actionButton: {
    width: ACTION_WIDTH,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: 4,
  },
  actionIcon: {
    fontSize: 20,
    marginBottom: 4,
  },
  actionLabel: {
    color: '#fff',
    fontSize: 11,
    fontWeight: '700',
    textAlign: 'center',
  },
});
