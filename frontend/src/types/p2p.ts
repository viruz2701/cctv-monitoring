export interface P2PDevice {
    id: number;
    serial: string;
    brand: string;
    status: 'online' | 'offline';
    lastSeen?: string;
}

export interface P2PRegistrationForm {
    serial: string;
    brand: string;
    securityCode: string;
    cloudUser?: string;
    cloudPass?: string;
}

export interface PTZCommand {
    command: 'left' | 'right' | 'up' | 'down' | 'zoom_in' | 'zoom_out';
    speed?: number;
}