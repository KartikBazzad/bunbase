import { useRef } from "react";
import { useTheme } from "../contexts/ThemeContext";
import { Sun, Moon, Palette } from "lucide-react";

const THEME_POPOVER_ID = "theme-switcher-popover";
const THEME_ANCHOR_NAME = "--theme-switcher-anchor";

export function ThemeSwitcher() {
  const { theme, setTheme } = useTheme();
  const popoverRef = useRef<HTMLUListElement>(null);

  const closePopover = () => {
    popoverRef.current?.hidePopover?.();
  };

  return (
    <>
      <button
        type="button"
        className="btn btn-ghost btn-circle"
        popovertarget={THEME_POPOVER_ID}
        style={{ anchorName: THEME_ANCHOR_NAME }}
        aria-label="Switch theme"
      >
        <Sun className="w-5 h-5" />
      </button>
      <ul
        ref={popoverRef}
        id={THEME_POPOVER_ID}
        popover="auto"
        role="menu"
        className="dropdown menu w-52 rounded-box bg-base-100 shadow-sm p-2 border border-base-300"
        style={{ positionAnchor: THEME_ANCHOR_NAME }}
      >
        <li>
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              setTheme("light");
              closePopover();
            }}
            className={theme === "light" ? "active" : ""}
          >
            <Sun className="w-4 h-4 mr-2 shrink-0" />
            Light
          </button>
        </li>
        <li>
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              setTheme("dark");
              closePopover();
            }}
            className={theme === "dark" ? "active" : ""}
          >
            <Moon className="w-4 h-4 mr-2 shrink-0" />
            Dark
          </button>
        </li>
        <li>
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              setTheme("cupcake");
              closePopover();
            }}
            className={theme === "cupcake" ? "active" : ""}
          >
            <Palette className="w-4 h-4 mr-2 shrink-0" />
            Cupcake
          </button>
        </li>
      </ul>
    </>
  );
}
