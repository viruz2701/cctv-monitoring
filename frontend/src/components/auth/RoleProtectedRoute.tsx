import React from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '../../hooks/useAuth';
import { UserRole } from '../../types';

interface RoleProtectedRouteProps {
    allowedRoles: UserRole[];
    redirectPath?: string;
}

export function RoleProtectedRoute({ allowedRoles, redirectPath = '/dashboard' }: RoleProtectedRouteProps) {
    const { user } = useAuth();

    if (!user) {
        return <Navigate to="/login" replace />;
    }

    if (!allowedRoles.includes(user.role)) {
        return <Navigate to={redirectPath} replace />;
    }

    return <Outlet />;
}
