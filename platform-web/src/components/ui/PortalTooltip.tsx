import { useState, useRef, useEffect, useCallback, ReactNode } from "react";
import { createPortal } from "react-dom";

const GAP = 8;
const Z_INDEX = 9999;

interface PortalTooltipProps {
  label: string;
  children: ReactNode;
}

export function PortalTooltip({ label, children }: PortalTooltipProps) {
  const [visible, setVisible] = useState(false);
  const triggerRef = useRef<HTMLDivElement>(null);

  const updatePosition = useCallback(() => {
    const el = triggerRef.current;
    if (!el || !visible) return {};
    const rect = el.getBoundingClientRect();
    return {
      position: "fixed" as const,
      left: rect.right + GAP,
      top: rect.top + rect.height / 2,
      transform: "translateY(-50%)",
      zIndex: Z_INDEX,
    };
  }, [visible]);

  const [style, setStyle] = useState<Record<string, string | number>>({});

  useEffect(() => {
    if (!visible) return;
    setStyle(updatePosition());
    const interval = setInterval(() => setStyle(updatePosition()), 100);
    return () => clearInterval(interval);
  }, [visible, updatePosition]);

  return (
    <>
      <div
        ref={triggerRef}
        className="inline-flex w-full"
        onMouseEnter={() => setVisible(true)}
        onMouseLeave={() => setVisible(false)}
        onFocus={() => setVisible(true)}
        onBlur={() => setVisible(false)}
      >
        {children}
      </div>
      {visible &&
        createPortal(
          <div
            role="tooltip"
            className="px-2 py-1.5 text-sm bg-base-100 border border-base-300 rounded shadow-lg whitespace-nowrap"
            style={style}
          >
            {label}
          </div>,
          document.body
        )}
    </>
  );
}
