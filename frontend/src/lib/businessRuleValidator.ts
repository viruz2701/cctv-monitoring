// businessRuleValidator — client-side валидация бизнес-правил.
//
// P1-2.4: Business Rule Validation
//   - SLA pause warning при status change
//   - Gatekeeper fail warning
//   - Checklist incomplete warning
//   - Inventory shortage warning

export type RuleSeverity = 'warning' | 'error' | 'info';

export interface BusinessRule {
    id: string;
    field: string;
    severity: RuleSeverity;
    message: string;
    detail?: string;
}

export type WorkOrderStatus = 'open' | 'in_progress' | 'paused' | 'completed' | 'cancelled' | 'rejected';
export type GatekeeperStep = 'photos' | 'checklist' | 'parts' | 'signature';
export type WOType = 'preventive' | 'corrective' | 'emergency' | 'routine' | 'inspection';

export interface WorkOrderFormState {
    status?: WorkOrderStatus;
    type?: WOType;
    checklistComplete?: boolean;
    gatekeeperStep?: GatekeeperStep;
    partsAvailable?: boolean;
    slaActive?: boolean;
    assignedTo?: string;
}

// Валидация смены статуса
function validateStatusChange(form: WorkOrderFormState): BusinessRule[] {
    const rules: BusinessRule[] = [];
    if (form.status === 'paused' && form.slaActive) {
        rules.push({
            id: 'sla-pause-warning',
            field: 'status',
            severity: 'warning',
            message: 'Pausing this work order will pause the SLA timer',
            detail: 'SLA will resume when work order is set back to In Progress',
        });
    }
    if (form.status === 'completed' && !form.checklistComplete) {
        rules.push({
            id: 'checklist-incomplete',
            field: 'checklist',
            severity: 'error',
            message: 'Checklist must be completed before closing',
        });
    }
    if (form.status === 'cancelled' && !form.assignedTo) {
        rules.push({
            id: 'cancel-no-assignee',
            field: 'status',
            severity: 'info',
            message: 'This work order has no assignee. Consider reassigning instead.',
        });
    }
    return rules;
}

// Валидация Gatekeeper
function validateGatekeeper(form: WorkOrderFormState): BusinessRule[] {
    const rules: BusinessRule[] = [];
    if (!form.gatekeeperStep) return rules;

    if (form.gatekeeperStep === 'checklist' && !form.checklistComplete) {
        rules.push({
            id: 'gatekeeper-checklist',
            field: 'gatekeeper',
            severity: 'error',
            message: 'Gatekeeper checklist must be completed before verification',
        });
    }
    if (form.gatekeeperStep === 'parts' && !form.partsAvailable) {
        rules.push({
            id: 'inventory-shortage',
            field: 'parts',
            severity: 'warning',
            message: 'Required parts are not in stock',
            detail: 'Contact warehouse to order missing parts',
        });
    }
    return rules;
}

// Emergency WO validation
function validateEmergencyWO(form: WorkOrderFormState): BusinessRule[] {
    const rules: BusinessRule[] = [];
    if (form.type === 'emergency' && !form.assignedTo) {
        rules.push({
            id: 'emergency-no-assignee',
            field: 'assignee',
            severity: 'error',
            message: 'Emergency work orders must be assigned immediately',
        });
    }
    return rules;
}

/**
 * validateWorkOrderForm — полная валидация формы Work Order.
 * Возвращает массив BusinessRule для отображения inline warnings.
 */
export function validateWorkOrderForm(form: WorkOrderFormState): BusinessRule[] {
    return [
        ...validateStatusChange(form),
        ...validateGatekeeper(form),
        ...validateEmergencyWO(form),
    ];
}

/**
 * hasBlockingErrors — проверяет, есть ли errors (не warnings) в списке правил.
 */
export function hasBlockingErrors(rules: BusinessRule[]): boolean {
    return rules.some(r => r.severity === 'error');
}

/**
 * getFieldRules — возвращает правила для конкретного поля.
 */
export function getFieldRules(rules: BusinessRule[], field: string): BusinessRule[] {
    return rules.filter(r => r.field === field);
}
