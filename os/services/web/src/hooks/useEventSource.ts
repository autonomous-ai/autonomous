import { useEffect, useRef } from "react";

type OnMessage = (ev: MessageEvent) => void;
type OnOpen = () => void;
type OnError = (ev: Event) => void;

// Opens an SSE connection gated on tab visibility. Each EventSource holds
// one persistent HTTP/1.1 connection slot (Chrome caps at 6 per origin),
// so a monitor page that subscribes to several streams + an MJPEG img
// quickly starves its own fetches and can't even reload. When the tab
// is backgrounded this closes the stream; reopens when visible again.
export function useEventSource(
  url: string | null,
  handlers: { onMessage?: OnMessage; onOpen?: OnOpen; onError?: OnError } = {},
  opts: { enabled?: boolean } = {},
) {
  const { enabled = true } = opts;
  const handlersRef = useRef(handlers);
  handlersRef.current = handlers;

  useEffect(() => {
    if (!enabled || !url) return;

    let es: EventSource | null = null;
    const open = () => {
      if (es !== null) return;
      es = new EventSource(url, { withCredentials: true });
      if (handlersRef.current.onOpen) es.addEventListener("open", () => handlersRef.current.onOpen?.());
      es.onmessage = (ev) => handlersRef.current.onMessage?.(ev);
      es.onerror = (ev) => handlersRef.current.onError?.(ev);
    };
    const close = () => {
      if (es !== null) {
        es.close();
        es = null;
      }
    };
    const onVisibility = () => {
      if (document.hidden) close();
      else open();
    };

    if (!document.hidden) open();
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      close();
    };
  }, [url, enabled]);
}
