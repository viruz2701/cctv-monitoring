import React from 'react';
import { useAuth } from '../../hooks/useAuth';
import { UserRole } from '../../types';

interface PermissionGuardProps {
    children: React.ReactNode;
    requiredRole?: UserRole | UserRole[]; // If provided, strictly checks these roles
    fallback?: React.ReactNode;
    requireManageTickets?: boolean; // Helper prop for ticket management
    requireManageUsers?: boolean;   // Helper prop for user management
}

export function PermissionGuard({
    children,
    requiredRole,
    fallback = null,
    requireManageTickets,
    requireManageUsers
}: PermissionGuardProps) {
    const { user } = useAuth();

    if (!user) return <>{fallback}</>;

    // Admin bypass: admins always have access, consistent with useAuth.hasPermission
    if (user.role === 'admin') return <>{children}</>;

    if (requireManageUsers) {
        return <>{fallback}</>;
    }

    if (requireManageTickets) {
        if (['viewer'].includes(user.role)) return <>{fallback}</>;
    }

    if (requiredRole) {
        const roles = Array.isArray(requiredRole) ? requiredRole : [requiredRole];
        if (!roles.includes(user.role)) {
            return <>{fallback}</>;
        }
    }

    return <>{children}</>;
}
