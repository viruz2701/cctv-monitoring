// ═══════════════════════════════════════════════════════════════════════
// P2-CHAT: Voice Note Recorder for Mobile
//
// Использует expo-av для записи голосовых заметок и отправляет их
// как voice-type сообщения в чат Work Order через API.
//
// Состояния:
//   - idle: ожидание записи
//   - recording: идёт запись
//   - recorded: запись завершена, можно отправить/удалить
//   - uploading: отправка на сервер
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useRef, useEffect, useCallback } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  Alert,
  Animated,
  Platform,
} from 'react-native';
import { Audio } from 'expo-av';
import * as FileSystem from 'expo-file-system';
import { Ionicons } from '@expo/vector-icons';
import { apiClient } from '../api/client';

type RecordingState = 'idle' | 'recording' | 'recorded' | 'uploading';

interface VoiceNoteRecorderProps {
  workOrderId: string;
  onSent?: () => void;
  onError?: (error: string) => void;
  theme?: 'light' | 'dark';
}

const MAX_RECORDING_DURATION = 120; // 2 minutes
const VOICE_NOTE_FOLDER = 'voice_notes';

export default function VoiceNoteRecorder({
  workOrderId,
  onSent,
  onError,
  theme = 'light',
}: VoiceNoteRecorderProps) {
  const [state, setState] = useState<RecordingState>('idle');
  const [duration, setDuration] = useState(0);
  const [recordingUri, setRecordingUri] = useState<string | null>(null);
  const [voiceNoteUri, setVoiceNoteUri] = useState<string | null>(null);

  const recordingRef = useRef<Audio.Recording | null>(null);
  const soundRef = useRef<Audio.Sound | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);
  const pulseAnim = useRef(new Animated.Value(1)).current;

  const isDark = theme === 'dark';
  const bgColor = isDark ? '#1a1a2e' : '#ffffff';
  const textColor = isDark ? '#e0e0e0' : '#333333';
  const accentColor = '#3b82f6';

  // ── Permission ──────────────────────────────────────────────────────────

  const requestPermission = useCallback(async (): Promise<boolean> => {
    try {
      const { granted } = await Audio.requestPermissionsAsync();
      if (!granted) {
        Alert.alert(
          'Permission Required',
          'Microphone access is needed to record voice notes.',
        );
        return false;
      }
      return true;
    } catch (err) {
      console.error('Failed to request audio permission', err);
      return false;
    }
  }, []);

  // ── Start recording ─────────────────────────────────────────────────────

  const startRecording = useCallback(async () => {
    const hasPermission = await requestPermission();
    if (!hasPermission) return;

    try {
      await Audio.setAudioModeAsync({
        allowsRecordingIOS: true,
        playsInSilentModeIOS: true,
        staysActiveInBackground: false,
        shouldDuckAndroid: true,
      });

      const { recording } = await Audio.Recording.createAsync(
        Audio.RecordingOptionsPresets.HIGH_QUALITY,
        onRecordingStatusUpdate,
        MAX_RECORDING_DURATION * 1000,
      );

      recordingRef.current = recording;
      setState('recording');
      setDuration(0);

      // Pulse animation
      Animated.loop(
        Animated.sequence([
          Animated.timing(pulseAnim, {
            toValue: 1.2,
            duration: 500,
            useNativeDriver: true,
          }),
          Animated.timing(pulseAnim, {
            toValue: 1,
            duration: 500,
            useNativeDriver: true,
          }),
        ]),
      ).start();

      // Timer
      timerRef.current = setInterval(() => {
        setDuration((prev) => {
          if (prev >= MAX_RECORDING_DURATION) {
            stopRecording();
            return prev;
          }
          return prev + 1;
        });
      }, 1000);
    } catch (err) {
      console.error('Failed to start recording', err);
      onError?.('Failed to start recording');
      setState('idle');
    }
  }, [requestPermission, onError, pulseAnim]);

  // ── Recording status update ─────────────────────────────────────────────

  const onRecordingStatusUpdate = (status: Audio.RecordingStatus) => {
    if (!status.isRecording) {
      // Recording was stopped externally
      if (recordingRef.current) {
        const uri = recordingRef.current.getURI();
        if (uri) {
          setRecordingUri(uri);
        }
      }
    }
  };

  // ── Stop recording ──────────────────────────────────────────────────────

  const stopRecording = useCallback(async () => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = undefined;
    }
    pulseAnim.setValue(1);

    try {
      if (recordingRef.current) {
        await recordingRef.current.stopAndUnloadAsync();
        const uri = recordingRef.current.getURI();
        if (uri) {
          setVoiceNoteUri(uri);
        }
        recordingRef.current = null;
      }

      await Audio.setAudioModeAsync({
        allowsRecordingIOS: false,
      });

      setState('recorded');
    } catch (err) {
      console.error('Failed to stop recording', err);
      setState('idle');
    }
  }, [pulseAnim]);

  // ── Cancel recording ────────────────────────────────────────────────────

  const cancelRecording = useCallback(async () => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = undefined;
    }
    pulseAnim.setValue(1);

    try {
      if (recordingRef.current) {
        await recordingRef.current.stopAndUnloadAsync();
        recordingRef.current = null;
      }
      if (soundRef.current) {
        await soundRef.current.unloadAsync();
        soundRef.current = null;
      }

      // Clean up temp file
      if (voiceNoteUri) {
        await FileSystem.deleteAsync(voiceNoteUri, { idempotent: true });
      }

      await Audio.setAudioModeAsync({
        allowsRecordingIOS: false,
      });

      setState('idle');
      setVoiceNoteUri(null);
      setRecordingUri(null);
      setDuration(0);
    } catch (err) {
      console.error('Failed to cancel recording', err);
      setState('idle');
    }
  }, [pulseAnim, voiceNoteUri]);

  // ── Playback ────────────────────────────────────────────────────────────

  const playVoiceNote = useCallback(async () => {
    if (!voiceNoteUri) return;

    try {
      const { sound } = await Audio.Sound.createAsync(
        { uri: voiceNoteUri },
        { shouldPlay: true },
      );
      soundRef.current = sound;

      sound.setOnPlaybackStatusUpdate((status) => {
        if (status.isLoaded && !status.isPlaying && status.didJustFinish) {
          sound.unloadAsync();
          soundRef.current = null;
        }
      });
    } catch (err) {
      console.error('Failed to play voice note', err);
    }
  }, [voiceNoteUri]);

  // ── Upload ──────────────────────────────────────────────────────────────

  const uploadVoiceNote = useCallback(async () => {
    if (!voiceNoteUri) return;

    setState('uploading');

    try {
      // Read file as base64
      const fileInfo = await FileSystem.getInfoAsync(voiceNoteUri);
      if (!fileInfo.exists) {
        throw new Error('Voice note file not found');
      }

      // Create form data for upload
      const formData = new FormData();
      // Type assertion required for React Native's FormData
      const fileData: any = {
        uri: voiceNoteUri,
        type: 'audio/m4a',
        name: `voice_${Date.now()}.m4a`,
      };
      formData.append('file', fileData);

      // Upload file
      const uploadResponse = await apiClient.post(
        `/work-orders/${workOrderId}/chat/upload`,
        formData,
        {
          headers: { 'Content-Type': 'multipart/form-data' },
          timeout: 30000,
        },
      );

      const attachment = uploadResponse.data;

      // Send voice message
      await apiClient.post(`/work-orders/${workOrderId}/chat`, {
        text: 'Voice note',
        message_type: 'voice',
        attachments: [attachment],
      });

      // Cleanup
      await FileSystem.deleteAsync(voiceNoteUri, { idempotent: true });
      setState('idle');
      setVoiceNoteUri(null);
      setDuration(0);

      onSent?.();
    } catch (err: any) {
      console.error('Failed to upload voice note', err);
      onError?.(err?.message || 'Failed to upload voice note');
      setState('recorded');
    }
  }, [voiceNoteUri, workOrderId, onSent, onError]);

  // ── Cleanup ─────────────────────────────────────────────────────────────

  useEffect(() => {
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
      if (recordingRef.current) {
        recordingRef.current.stopAndUnloadAsync().catch(() => {});
      }
      if (soundRef.current) {
        soundRef.current.unloadAsync().catch(() => {});
      }
    };
  }, []);

  // ── Format duration ─────────────────────────────────────────────────────

  const formatDuration = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  // ── Render ──────────────────────────────────────────────────────────────

  return (
    <View style={[styles.container, { backgroundColor: bgColor }]}>
      {/* State: idle */}
      {state === 'idle' && (
        <TouchableOpacity
          style={[styles.button, { backgroundColor: accentColor }]}
          onPress={startRecording}
          accessibilityLabel="Start recording voice note"
          accessibilityRole="button"
        >
          <Ionicons name="mic" size={24} color="#fff" />
          <Text style={styles.buttonText}>Voice Note</Text>
        </TouchableOpacity>
      )}

      {/* State: recording */}
      {state === 'recording' && (
        <View style={styles.recordingContainer}>
          <Animated.View
            style={[
              styles.pulseCircle,
              { transform: [{ scale: pulseAnim }] },
            ]}
          >
            <TouchableOpacity
              style={[styles.stopButton, { backgroundColor: '#ef4444' }]}
              onPress={stopRecording}
              accessibilityLabel="Stop recording"
              accessibilityRole="button"
            >
              <Ionicons name="stop" size={24} color="#fff" />
            </TouchableOpacity>
          </Animated.View>

          <Text style={[styles.duration, { color: textColor }]}>
            {formatDuration(duration)}
          </Text>

          <TouchableOpacity
            style={styles.cancelButton}
            onPress={cancelRecording}
            accessibilityLabel="Cancel recording"
            accessibilityRole="button"
          >
            <Ionicons name="trash-outline" size={20} color="#ef4444" />
            <Text style={styles.cancelText}>Cancel</Text>
          </TouchableOpacity>
        </View>
      )}

      {/* State: recorded */}
      {state === 'recorded' && (
        <View style={styles.recordedContainer}>
          <Text style={[styles.duration, { color: textColor }]}>
            {formatDuration(duration)}
          </Text>

          <View style={styles.actionRow}>
            <TouchableOpacity
              style={styles.playButton}
              onPress={playVoiceNote}
              accessibilityLabel="Play voice note"
              accessibilityRole="button"
            >
              <Ionicons name="play-circle" size={32} color={accentColor} />
            </TouchableOpacity>

            <TouchableOpacity
              style={[styles.sendButton, { backgroundColor: accentColor }]}
              onPress={uploadVoiceNote}
              accessibilityLabel="Send voice note"
              accessibilityRole="button"
            >
              <Ionicons name="send" size={20} color="#fff" />
              <Text style={styles.buttonText}>Send</Text>
            </TouchableOpacity>

            <TouchableOpacity
              style={styles.cancelButton}
              onPress={cancelRecording}
              accessibilityLabel="Discard voice note"
              accessibilityRole="button"
            >
              <Ionicons name="close-circle" size={32} color="#ef4444" />
            </TouchableOpacity>
          </View>
        </View>
      )}

      {/* State: uploading */}
      {state === 'uploading' && (
        <View style={styles.uploadingContainer}>
          <Ionicons name="cloud-upload" size={24} color={accentColor} />
          <Text style={[styles.uploadingText, { color: textColor }]}>
            Uploading...
          </Text>
        </View>
      )}
    </View>
  );
}

// ── Styles ─────────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  container: {
    padding: 12,
    borderRadius: 12,
    alignItems: 'center',
    elevation: 2,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.1,
    shadowRadius: 4,
  },
  button: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    paddingVertical: 12,
    paddingHorizontal: 20,
    borderRadius: 24,
  },
  buttonText: {
    color: '#fff',
    fontSize: 14,
    fontWeight: '600',
  },
  recordingContainer: {
    alignItems: 'center',
    gap: 12,
  },
  pulseCircle: {
    padding: 4,
  },
  stopButton: {
    width: 48,
    height: 48,
    borderRadius: 24,
    alignItems: 'center',
    justifyContent: 'center',
  },
  duration: {
    fontSize: 18,
    fontWeight: '600',
    fontVariant: ['tabular-nums'],
  },
  recordedContainer: {
    alignItems: 'center',
    gap: 8,
  },
  actionRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 16,
  },
  playButton: {
    padding: 4,
  },
  sendButton: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingVertical: 10,
    paddingHorizontal: 20,
    borderRadius: 24,
  },
  cancelButton: {
    padding: 4,
    alignItems: 'center',
  },
  cancelText: {
    color: '#ef4444',
    fontSize: 12,
    marginTop: 2,
  },
  uploadingContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  uploadingText: {
    fontSize: 14,
  },
});
