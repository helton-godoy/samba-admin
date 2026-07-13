import { useEffect, useRef, useState } from 'react';
import { api } from '../api/client';
import type { Job } from '../api/generated';

const LAST_EVENT_ID_STORAGE_KEY = 'samba-admin:last-job-event-id';
const MAX_RECONNECT_ATTEMPTS = 4;
const MAX_SEEN_EVENTS = 250;
const STABLE_CONNECTION_MS = 15_000;

export type SseConnectionState = 'connecting' | 'connected' | 'reconnecting' | 'fallback' | 'unsupported';

export interface JobStreamEvent {
  id: string;
  type: string;
  job: Job;
  correlationId?: string;
  timestamp?: string;
}

export interface UseJobEventsOptions {
  enabled?: boolean;
  onJobEvent?(event: JobStreamEvent): void;
}

function storedLastEventId(): string | undefined {
  if (typeof window === 'undefined') return undefined;
  try {
    return window.sessionStorage.getItem(LAST_EVENT_ID_STORAGE_KEY) || undefined;
  } catch {
    return undefined;
  }
}

function storeLastEventId(id: string): void {
  if (!id || typeof window === 'undefined') return;
  try {
    window.sessionStorage.setItem(LAST_EVENT_ID_STORAGE_KEY, id);
  } catch {
    // Event recovery remains best effort when storage is disabled.
  }
}

function jobFromPayload(payload: unknown): Job | undefined {
  if (!payload || typeof payload !== 'object') return undefined;
  const candidate = payload as Record<string, unknown>;
  const job = (candidate.payload && typeof candidate.payload === 'object' ? candidate.payload : candidate) as Record<string, unknown>;
  if (typeof job.id !== 'string' || typeof job.status !== 'string') return undefined;
  return job as unknown as Job;
}

function eventIdentity(event: MessageEvent<string>, payload: Record<string, unknown>, type: string, job: Job): string {
  if (event.lastEventId) return event.lastEventId;
  const timestamp = typeof payload.timestamp === 'string' ? payload.timestamp : '';
  const resource = typeof payload.resourceId === 'string' ? payload.resourceId : job.id;
  return `${type}:${resource}:${timestamp}:${job.status}:${job.progress}`;
}

export function useJobEvents({ enabled = true, onJobEvent }: UseJobEventsOptions = {}) {
  const [connectionState, setConnectionState] = useState<SseConnectionState>(() => {
    return typeof EventSource === 'undefined' ? 'unsupported' : 'connecting';
  });
  const [reconnectAttempt, setReconnectAttempt] = useState(0);
  const callbackRef = useRef(onJobEvent);

  useEffect(() => {
    callbackRef.current = onJobEvent;
  }, [onJobEvent]);

  useEffect(() => {
    if (!enabled) return;
    if (typeof EventSource === 'undefined') {
      setConnectionState('unsupported');
      return;
    }

    let disposed = false;
    let source: EventSource | undefined;
    let reconnectTimer: number | undefined;
    let stabilityTimer: number | undefined;
    let attempts = 0;
    let connect: () => void = () => undefined;
    const seen = new Set<string>();
    let lastEventId = storedLastEventId();

    const remember = (id: string) => {
      seen.add(id);
      if (seen.size > MAX_SEEN_EVENTS) {
        const first = seen.values().next().value as string | undefined;
        if (first) seen.delete(first);
      }
    };

    const scheduleReconnect = () => {
      if (disposed) return;
      attempts += 1;
      setReconnectAttempt(attempts);
      if (attempts > MAX_RECONNECT_ATTEMPTS) {
        setConnectionState('fallback');
        return;
      }
      setConnectionState('reconnecting');
      const delay = Math.min(1_000 * 2 ** (attempts - 1), 30_000);
      reconnectTimer = window.setTimeout(connect, delay);
    };

    const handleEvent = (message: Event) => {
      const event = message as MessageEvent<string>;
      let payload: Record<string, unknown>;
      try {
        payload = JSON.parse(event.data) as Record<string, unknown>;
      } catch {
        return;
      }
      const type = typeof payload.type === 'string' ? payload.type : event.type;
      const job = jobFromPayload(payload);
      if (!job || !type.startsWith('job.')) return;
      const id = eventIdentity(event, payload, type, job);
      if (seen.has(id)) return;
      remember(id);
      if (event.lastEventId) {
        lastEventId = event.lastEventId;
        storeLastEventId(lastEventId);
      }
      callbackRef.current?.({
        id,
        type,
        job,
        correlationId: typeof payload.correlationId === 'string' ? payload.correlationId : undefined,
        timestamp: typeof payload.timestamp === 'string' ? payload.timestamp : undefined
      });
    };

    const eventTypes = ['job.created', 'job.updated', 'job.completed', 'job.failed', 'job.cancelled'];

    connect = () => {
      if (disposed) return;
      setConnectionState(attempts === 0 ? 'connecting' : 'reconnecting');
      source?.close();
      const activeSource = new EventSource(api.eventsUrl(lastEventId), { withCredentials: true });
      source = activeSource;
      activeSource.onopen = () => {
        setConnectionState('connected');
        // Uma conexão que abre e cai imediatamente continua contando como
        // falha. Só zeramos o backoff depois de um período estável.
        if (stabilityTimer !== undefined) window.clearTimeout(stabilityTimer);
        stabilityTimer = window.setTimeout(() => {
          if (disposed || source !== activeSource) return;
          attempts = 0;
          setReconnectAttempt(0);
        }, STABLE_CONNECTION_MS);
      };
      activeSource.onmessage = handleEvent;
      eventTypes.forEach((eventType) => activeSource.addEventListener(eventType, handleEvent));
      activeSource.onerror = () => {
        if (source !== activeSource) return;
        if (stabilityTimer !== undefined) window.clearTimeout(stabilityTimer);
        activeSource.close();
        scheduleReconnect();
      };
    };

    connect();
    return () => {
      disposed = true;
      source?.close();
      if (reconnectTimer !== undefined) window.clearTimeout(reconnectTimer);
      if (stabilityTimer !== undefined) window.clearTimeout(stabilityTimer);
    };
  }, [enabled]);

  return {
    connectionState,
    reconnectAttempt,
    shouldPoll: connectionState === 'fallback' || connectionState === 'unsupported'
  };
}
