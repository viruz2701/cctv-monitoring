import React, { useMemo } from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { WorkOrder } from '../types';
import { formatWorkOrderDate, formatSLADeadline, isSLAPast } from '../utils/dateHelpers';

interface Props {
  workOrder: WorkOrder;
  onPress: () => void;
}

function WorkOrderCardInner({ workOrder, onPress }: Props) {
  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'critical':
        return { bg: '#fef2f2', text: '#dc2626', border: '#fecaca' };
      case 'high':
        return { bg: '#fff7ed', text: '#ea580c', border: '#fed7aa' };
      case 'medium':
        return { bg: '#fefce8', text: '#ca8a04', border: '#fde68a' };
      default:
        return { bg: '#f0fdf4', text: '#16a34a', border: '#bbf7d0' };
    }
  };

  const getStatusLabel = (status: string): string => {
    switch (status) {
      case 'open':
        return 'Открыт';
      case 'in_progress':
        return 'В работе';
      case 'completed':
        return 'Завершён';
      case 'cancelled':
        return 'Отменён';
      default:
        return status;
    }
  };

  const getTypeLabel = (type: string): string => {
    switch (type) {
      case 'preventive':
        return 'Плановое ТО';
      case 'corrective':
        return 'Ремонт';
      case 'emergency':
        return 'Аварийный';
      default:
        return type;
    }
  };

  const priorityColors = useMemo(
    () => getPriorityColor(workOrder.priority),
    [workOrder.priority],
  );
  const slaOverdue = useMemo(
    () => (workOrder.sla_deadline ? isSLAPast(workOrder.sla_deadline) : false),
    [workOrder.sla_deadline],
  );
  const formattedDate = useMemo(
    () => formatWorkOrderDate(workOrder.created_at),
    [workOrder.created_at],
  );
  const statusLabel = useMemo(
    () => getStatusLabel(workOrder.status),
    [workOrder.status],
  );
  const typeLabel = useMemo(
    () => getTypeLabel(workOrder.type),
    [workOrder.type],
  );

  return (
    <TouchableOpacity style={styles.card} onPress={onPress} activeOpacity={0.7}>
      <View style={styles.header}>
        <View
          style={[
            styles.priorityBadge,
            {
              backgroundColor: priorityColors.bg,
              borderColor: priorityColors.border,
            },
          ]}
        >
          <Text style={[styles.priorityText, { color: priorityColors.text }]}>
            {workOrder.priority.toUpperCase()}
          </Text>
        </View>
        <Text style={styles.type}>{typeLabel}</Text>
      </View>

      <Text style={styles.title} numberOfLines={2}>
        {workOrder.device_name || workOrder.device_id}
      </Text>

      {workOrder.site_name && (
        <Text style={styles.site} numberOfLines={1}>
          {workOrder.site_name}
        </Text>
      )}

      <View style={styles.footer}>
        <Text style={styles.status}>{statusLabel}</Text>
        <Text style={styles.date}>{formattedDate}</Text>
      </View>

      {workOrder.sla_deadline && (
        <View style={[styles.sla, slaOverdue && styles.slaOverdue]}>
          <Text style={[styles.slaText, slaOverdue && styles.slaTextOverdue]}>
            SLA: {formatSLADeadline(workOrder.sla_deadline)}
          </Text>
        </View>
      )}
    </TouchableOpacity>
  );
}

// ═══ P3-UI.3: React.memo для оптимизации FlatList ═══
// Предотвращает лишние ре-рендеры при скролле списка
const WorkOrderCard = React.memo(WorkOrderCardInner);
export default WorkOrderCard;

const styles = StyleSheet.create({
  card: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.05,
    shadowRadius: 4,
    elevation: 2,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  priorityBadge: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 6,
    borderWidth: 1,
  },
  priorityText: {
    fontSize: 10,
    fontWeight: '700',
  },
  type: {
    fontSize: 12,
    color: '#64748b',
  },
  title: {
    fontSize: 16,
    fontWeight: '600',
    color: '#1e293b',
    marginBottom: 4,
  },
  site: {
    fontSize: 14,
    color: '#64748b',
    marginBottom: 12,
  },
  footer: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  status: {
    fontSize: 13,
    fontWeight: '500',
    color: '#2563eb',
  },
  date: {
    fontSize: 12,
    color: '#94a3b8',
  },
  sla: {
    marginTop: 8,
    paddingTop: 8,
    borderTopWidth: 1,
    borderTopColor: '#f1f5f9',
  },
  slaOverdue: {
    borderTopColor: '#fecaca',
  },
  slaText: {
    fontSize: 11,
    color: '#dc2626',
    fontWeight: '500',
  },
  slaTextOverdue: {
    color: '#991b1b',
  },
});