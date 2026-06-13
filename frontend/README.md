# CCTV Health Monitor

A modern, responsive dashboard for monitoring CCTV camera health, managing tickets, and tracking system alerts. Built with React, TypeScript, and Tailwind CSS.

## 🚀 Features

### 📊 Interactive Dashboard
- **Real-time Stats**: View total devices, online/offline status, and health metrics at a glance.
- **Dynamic Charts**: Visualizations for resolution rates and response times.
- **Recent Alerts**: Live feed of critical system events.

### 🎫 Ticketing System
- **Full Lifecycle Management**: Create, track, and update tickets (Open -> In Progress -> Resolved -> Closed).
- **Detail View**: Rich ticket details with comment history, priority badges, and assignee tracking.
- **Filtering**: Filter tickets by status and priority.

### 🔔 Alert Management
- **System Alerts**: Monitor hardware and network issues (`Camera Offline`, `HDD Error`, `Network Latency`).
- **Actionable Workflow**: Acknowledge and resolve alerts directly from the UI.
- **Severity Levels**: Visual indicators for Critical, Warning, and Info alerts.

### 👥 User Management & RBAC
- **Role-Based Access Control**:
  - **Admin**: Full access to all features.
  - **Manager**: Can manage tickets and alerts.
  - **Technician**: Can view and update assigned tickets.
  - **Viewer**: Read-only access.
- **User Administration**: Add, edit, and manage user roles and permissions.

### 🎨 Modern UI/UX
- **Dark Mode**: Fully supported dark theme with seamless switching.
- **Responsive Design**: Optimized for desktop, tablet, and mobile viewing.
- **Premium Aesthetics**: Glassmorphism effects, smooth transitions, and polished components.

## 🛠️ Tech Stack

- **Framework**: [React 18](https://reactjs.org/) + [Vite](https://vitejs.dev/)
- **Language**: [TypeScript](https://www.typescriptlang.org/)
- **Styling**: [Tailwind CSS](https://tailwindcss.com/)
- **Icons**: [Lucide React](https://lucide.dev/)
- **State Management**: React Context API (`DataContext`)
- **Routing**: [React Router v6](https://reactrouter.com/)

## 📦 Installation

1.  **Clone the repository**
    ```bash
    git clone https://github.com/mananmaheshwari1702/CCTV-Health-Monitor.git
    cd CCTV-Health-Monitor
    ```

2.  **Install dependencies**
    ```bash
    npm install
    ```

3.  **Start the development server**
    ```bash
    npm run dev
    ```

4.  **Open in browser**
    Navigate to `http://localhost:5173`

## 🏗️ Project Structure

```
src/
├── components/         # Reusable UI components
│   ├── auth/          # Authentication guards
│   ├── layout/        # Sidebar, Header, Layout wrapper
│   └── ui/            # Atomic components (Card, Button, Badge, etc.)
├── context/           # Global state (DataContext, ThemeContext)
├── data/              # Mock data for prototyping
├── hooks/             # Custom hooks (useAuth, useData)
├── pages/             # Route pages (Dashboard, Tickets, Alerts, etc.)
└── types/             # TypeScript interfaces and types
```

## 🔐 Credentials (Prototype)

The application uses a simulated backend. You can test different roles by switching users in the mock authentication flow or checking the `src/data/mockData.ts` file for available user profiles.

- **Admin User**: Full access
- **Tech User**: Restricted access

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
