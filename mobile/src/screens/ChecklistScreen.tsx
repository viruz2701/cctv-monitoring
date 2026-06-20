import React, { useState } from 'react';
import { View, Text, FlatList, TouchableOpacity, StyleSheet } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { NativeStackScreenProps, NativeStackNavigationProp } from '@react-navigation/native-stack';
import { RootStackParamList, ChecklistItem } from '../types';

type Props = NativeStackScreenProps<RootStackParamList, 'Checklist'>;

export default function ChecklistScreen({ route }: Props) {
  const { workOrder } = route.params;
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();

  const [checklist, setChecklist] = useState<ChecklistItem[]>(
    workOrder.checklist || [],
  );

  const toggleItem = (index: number) => {
    const updated = [...checklist];
    updated[index] = { ...updated[index], completed: !updated[index].completed };
    setChecklist(updated);
  };

  const completedCount = checklist.filter((item) => item.completed).length;
  const progress = checklist.length > 0 ? (completedCount / checklist.length) * 100 : 0;

  const handleContinue = () => {
    navigation.navigate('PhotoCapture', {
      workOrder,
      checklist,
    });
  };

  return (
    <View style={styles.container}>
      <View style={styles.progressContainer}>
        <View style={styles.progressHeader}>
          <Text style={styles.progressText}>
            Прогресс: {completedCount}/{checklist.length}
          </Text>
          <Text style={styles.progressPercent}>{Math.round(progress)}%</Text>
        </View>
        <View style={styles.progressBar}>
          <View style={[styles.progressFill, { width: `${progress}%` }]} />
        </View>
      </View>

      <FlatList
        data={checklist}
        keyExtractor={(_, index) => index.toString()}
        renderItem={({ item, index }) => (
          <TouchableOpacity
            style={[styles.item, item.completed && styles.itemCompleted]}
            onPress={() => toggleItem(index)}
            activeOpacity={0.7}
          >
            <View style={[styles.checkbox, item.completed && styles.checkboxChecked]}>
              {item.completed && <Text style={styles.checkmark}>✓</Text>}
            </View>
            <Text style={[styles.itemText, item.completed && styles.itemTextCompleted]}>
              {item.task}
            </Text>
          </TouchableOpacity>
        )}
        contentContainerStyle={styles.list}
        ListEmptyComponent={
          <View style={styles.empty}>
            <Text style={styles.emptyText}>Нет пунктов чек-листа</Text>
          </View>
        }
      />

      <TouchableOpacity
        style={[styles.button, progress < 100 && styles.buttonDisabled]}
        onPress={handleContinue}
        disabled={progress < 100}
        activeOpacity={0.7}
      >
        <Text style={styles.buttonText}>Продолжить → Фотофиксация</Text>
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f1f5f9',
  },
  progressContainer: {
    backgroundColor: '#fff',
    padding: 16,
    marginBottom: 8,
  },
  progressHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  progressText: {
    fontSize: 14,
    fontWeight: '500',
    color: '#1e293b',
  },
  progressPercent: {
    fontSize: 14,
    fontWeight: '600',
    color: '#2563eb',
  },
  progressBar: {
    height: 8,
    backgroundColor: '#e2e8f0',
    borderRadius: 4,
    overflow: 'hidden',
  },
  progressFill: {
    height: '100%',
    backgroundColor: '#2563eb',
    borderRadius: 4,
  },
  list: {
    padding: 16,
  },
  item: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#fff',
    padding: 16,
    borderRadius: 8,
    marginBottom: 8,
  },
  itemCompleted: {
    backgroundColor: '#f0fdf4',
  },
  checkbox: {
    width: 24,
    height: 24,
    borderRadius: 6,
    borderWidth: 2,
    borderColor: '#cbd5e1',
    marginRight: 12,
    justifyContent: 'center',
    alignItems: 'center',
  },
  checkboxChecked: {
    backgroundColor: '#16a34a',
    borderColor: '#16a34a',
  },
  checkmark: {
    color: '#fff',
    fontSize: 16,
    fontWeight: 'bold',
  },
  itemText: {
    flex: 1,
    fontSize: 15,
    color: '#1e293b',
  },
  itemTextCompleted: {
    textDecorationLine: 'line-through',
    color: '#64748b',
  },
  empty: {
    padding: 40,
    alignItems: 'center',
  },
  emptyText: {
    fontSize: 14,
    color: '#94a3b8',
  },
  button: {
    backgroundColor: '#2563eb',
    margin: 16,
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
  },
  buttonDisabled: {
    backgroundColor: '#cbd5e1',
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});