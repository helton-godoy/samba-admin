import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useJobEvents } from '../hooks/useJobEvents';

class FlappingEventSource {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSED = 2;

  readonly CONNECTING = 0;
  readonly OPEN = 1;
  readonly CLOSED = 2;
  readonly url: string;
  readonly withCredentials = true;
  readyState = FlappingEventSource.CONNECTING;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;

  constructor(url: string | URL) {
    this.url = String(url);
    window.setTimeout(() => {
      this.readyState = FlappingEventSource.OPEN;
      this.onopen?.(new Event('open'));
      window.setTimeout(() => {
        this.readyState = FlappingEventSource.CLOSED;
        this.onerror?.(new Event('error'));
      }, 1);
    }, 0);
  }

  close() {
    this.readyState = FlappingEventSource.CLOSED;
  }

  addEventListener() {}
  removeEventListener() {}
  dispatchEvent() { return true; }
}

describe('Reconexão SSE', () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('migra para polling após quedas curtas repetidas', async () => {
    vi.useFakeTimers();
    vi.stubGlobal('EventSource', FlappingEventSource as unknown as typeof EventSource);

    const { result } = renderHook(() => useJobEvents());
    await act(async () => {
      await vi.advanceTimersByTimeAsync(20_000);
    });

    expect(result.current.connectionState).toBe('fallback');
    expect(result.current.shouldPoll).toBe(true);
  });
});
