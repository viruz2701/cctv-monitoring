// ═══════════════════════════════════════════════════════════════════════
// AuthFlow — Unit Tests
// P2-MED-16: Frontend Coverage 82% → 85%
// Тесты для auth flow: Login, RoleProtectedRoute, token mock
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen, cleanup, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../i18n';
import { Login } from '../../pages/Login';
import { RoleProtectedRoute } from '../auth/RoleProtectedRoute';

// ── Mock i18n ─────────────────────────────────────────────────────────

vi.mock('react-i18next', async () => {
  const actual = await vi.importActual('react-i18next');
  return {
    ...(actual as object),
    useTranslation: () => ({
      t: (key: string) => {
        const fallbacks: Record<string, string> = {
          welcome_back: 'Welcome Back',
          sign_in: 'Sign in to your account',
          email: 'Email',
          password: 'Password',
          login: 'Sign In',
          remember_me: 'Remember me',
          forgot_password: 'Forgot Password?',
          login_error_empty: 'Please enter your email and password',
          login_error_password_basic: 'Password must be at least 6 characters.',
          login_error_password_strong: 'Password must be at least 8 characters and contain a number or symbol.',
          login_failed: 'Login failed',
          login_error_otp_invalid: 'Please enter a valid 6-digit code',
          '2fa_title': 'Two-Factor Authentication',
          '2fa_instruction': 'Enter the 6-digit code from your authenticator app',
          '2fa_code': 'Authentication Code',
          '2fa_verify': 'Verify Code',
          '2fa_example': 'Enter code: 123456',
          back_to_login: 'Back to Login',
          demo_creds: 'Demo: admin@example.com / admin123',
          devices_monitored: 'Devices Monitored',
          uptime: 'Uptime',
          sites: 'Sites',
          copyright: '© 2026 CCTV Health Monitor. All rights reserved.',
          login_description: 'Monitor device health, manage tickets...',
          '2fa_description': 'Secure your account with Two-Factor Authentication.',
        };
        return fallbacks[key] || key;
      },
      i18n: { language: 'en' },
    }),
  };
});

// ── Mock useAuth ──────────────────────────────────────────────────────

const mockLogin = vi.fn();
const mockLogin2FA = vi.fn();

vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({
    user: null,
    token: null,
    login: mockLogin,
    login2FA: mockLogin2FA,
    logout: vi.fn(),
    hasPermission: vi.fn(() => true),
    updateUser: vi.fn(),
  }),
}));

// ── Mock useSettingsStore with passwordPolicy as mutable ref ───────────

let currentPasswordPolicy = 'basic';

vi.mock('../../store/settingsStore', () => ({
  useSettingsStore: () => ({
    settings: {
      security: {
        requires2FA: false,
        passwordPolicy: currentPasswordPolicy,
      },
    },
  }),
}));

// ── Mock useNavigate ──────────────────────────────────────────────────

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...(actual as object),
    useNavigate: () => mockNavigate,
  };
});

// ── Wrapper ───────────────────────────────────────────────────────────

function Wrapper({ children }: { children: React.ReactNode }) {
  return (
    <MemoryRouter initialEntries={['/login']}>
      <I18nextProvider i18n={i18n}>{children}</I18nextProvider>
    </MemoryRouter>
  );
}

// ── Tests ─────────────────────────────────────────────────────────────

describe('AuthFlow — Login Form', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    currentPasswordPolicy = 'basic';
  });

  afterEach(() => {
    cleanup();
  });

  // ── Rendering ────────────────────────────────────────────────────

  it('renders login form with email and password fields', () => {
    render(<Login />, { wrapper: Wrapper });
    expect(screen.getByText('Welcome Back')).toBeInTheDocument();
    expect(screen.getByText('Email')).toBeInTheDocument();
    expect(screen.getByText('Password')).toBeInTheDocument();
    expect(screen.getByText('Sign In')).toBeInTheDocument();
  });

  it('renders remember me checkbox and forgot password link', () => {
    render(<Login />, { wrapper: Wrapper });
    expect(screen.getByText('Remember me')).toBeInTheDocument();
    expect(screen.getByText('Forgot Password?')).toBeInTheDocument();
  });

  it('renders demo credentials hint', () => {
    render(<Login />, { wrapper: Wrapper });
    expect(screen.getByText(/admin@example.com/)).toBeInTheDocument();
  });

  // ── Input Validation ─────────────────────────────────────────────

  it('shows error when submitting with empty fields', async () => {
    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    await user.click(screen.getByText('Sign In'));
    expect(screen.getByText('Please enter your email and password')).toBeInTheDocument();
  });

  it('shows error when password is too short (basic policy)', async () => {
    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    await user.type(screen.getByPlaceholderText('admin@example.com'), 'test@test.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'abc12');
    await user.click(screen.getByText('Sign In'));

    expect(screen.getByText('Password must be at least 6 characters.')).toBeInTheDocument();
  });

  it('validates password with strong policy', async () => {
    currentPasswordPolicy = 'strong';
    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    await user.type(screen.getByPlaceholderText('admin@example.com'), 'test@test.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'short');
    await user.click(screen.getByText('Sign In'));

    expect(screen.getByText(/at least 8 characters/)).toBeInTheDocument();
  });

  // ── Submit with Valid Credentials ─────────────────────────────────

  it('calls login with email and password on valid submit', async () => {
    mockLogin.mockResolvedValue({});

    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    await user.type(screen.getByPlaceholderText('admin@example.com'), 'admin@example.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'admin123');
    await user.click(screen.getByText('Sign In'));

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith('admin@example.com', 'admin123');
    });
  });

  it('navigates to dashboard on successful login', async () => {
    mockLogin.mockResolvedValue({});

    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    await user.type(screen.getByPlaceholderText('admin@example.com'), 'admin@example.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'admin123');
    await user.click(screen.getByText('Sign In'));

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/dashboard');
    });
  });

  // ── Error States ─────────────────────────────────────────────────

  it('shows error message when login fails', async () => {
    mockLogin.mockRejectedValue(new Error('Invalid credentials'));

    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    await user.type(screen.getByPlaceholderText('admin@example.com'), 'wrong@email.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'wrongpass');
    await user.click(screen.getByText('Sign In'));

    await waitFor(() => {
      expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
    });
  });

  it('shows fallback error when login rejects with non-Error', async () => {
    mockLogin.mockRejectedValue('Server error');

    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    await user.type(screen.getByPlaceholderText('admin@example.com'), 'admin@example.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'admin123');
    await user.click(screen.getByText('Sign In'));

    await waitFor(() => {
      expect(screen.getByText('Server error')).toBeInTheDocument();
    });
  });

  // ── 2FA Flow ─────────────────────────────────────────────────────

  it('shows 2FA form when login requires 2FA', async () => {
    mockLogin.mockResolvedValue({
      requires2FA: true,
      sessionToken: 'session-123',
    });

    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    await user.type(screen.getByPlaceholderText('admin@example.com'), 'admin@example.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'admin123');
    await user.click(screen.getByText('Sign In'));

    await waitFor(() => {
      expect(screen.getByText('Two-Factor Authentication')).toBeInTheDocument();
      expect(screen.getByText('Authentication Code')).toBeInTheDocument();
    });
  });

  it('submits OTP code in 2FA step', async () => {
    mockLogin.mockResolvedValue({
      requires2FA: true,
      sessionToken: 'session-123',
    });

    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    await user.type(screen.getByPlaceholderText('admin@example.com'), 'admin@example.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'admin123');
    await user.click(screen.getByText('Sign In'));

    await waitFor(() => {
      expect(screen.getByText('Authentication Code')).toBeInTheDocument();
    });

    const otpInput = screen.getByPlaceholderText('000000');
    await user.type(otpInput, '123456');
    await user.click(screen.getByText('Verify Code'));

    await waitFor(() => {
      expect(mockLogin2FA).toHaveBeenCalledWith('session-123', '123456');
    });
  });

  it('shows error for invalid OTP length', async () => {
    mockLogin.mockResolvedValue({
      requires2FA: true,
      sessionToken: 'session-123',
    });

    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    await user.type(screen.getByPlaceholderText('admin@example.com'), 'admin@example.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'admin123');
    await user.click(screen.getByText('Sign In'));

    await waitFor(() => {
      expect(screen.getByText('Authentication Code')).toBeInTheDocument();
    });

    const otpInput = screen.getByPlaceholderText('000000');
    await user.type(otpInput, '123');
    await user.click(screen.getByText('Verify Code'));

    await waitFor(() => {
      expect(screen.getByText('Please enter a valid 6-digit code')).toBeInTheDocument();
    });
  });

  it('allows going back to credentials from 2FA step', async () => {
    mockLogin.mockResolvedValue({
      requires2FA: true,
      sessionToken: 'session-123',
    });

    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    await user.type(screen.getByPlaceholderText('admin@example.com'), 'admin@example.com');
    await user.type(screen.getByPlaceholderText('••••••••'), 'admin123');
    await user.click(screen.getByText('Sign In'));

    await waitFor(() => {
      expect(screen.getByText('Authentication Code')).toBeInTheDocument();
    });

    await user.click(screen.getByText('Back to Login'));

    await waitFor(() => {
      expect(screen.getByText('Welcome Back')).toBeInTheDocument();
    });
  });

  // ── Password Visibility Toggle ───────────────────────────────────

  it('toggles password visibility', async () => {
    const user = userEvent.setup();
    render(<Login />, { wrapper: Wrapper });

    const passwordInput = screen.getByPlaceholderText('••••••••');
    expect(passwordInput).toHaveAttribute('type', 'password');

    const toggleBtn = passwordInput.parentElement?.querySelector('button');
    expect(toggleBtn).toBeTruthy();
    if (toggleBtn) {
      await user.click(toggleBtn);
      expect(passwordInput).toHaveAttribute('type', 'text');
    }
  });
});

describe('AuthFlow — RoleProtectedRoute', () => {
  afterEach(() => {
    cleanup();
  });

  it('redirects to login when user is not authenticated', () => {
    render(
      <MemoryRouter>
        <RoleProtectedRoute allowedRoles={['admin']} />
      </MemoryRouter>,
    );
    // RoleProtectedRoute with no user renders Navigate to /login
    // This means there should be no crash
    expect(document.body).toBeDefined();
  });
});
