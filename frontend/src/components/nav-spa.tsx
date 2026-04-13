import type { ButtonHTMLAttributes, KeyboardEvent, ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";

type NavButtonProps = {
  readonly to: string;
  readonly className?: string;
  readonly children: ReactNode;
} & Omit<ButtonHTMLAttributes<HTMLButtonElement>, "onClick" | "type">;

/** Same-tab SPA navigation; avoids Mac ⌘-click / middle-click opening a new browser tab on `<a href>`. */
export function NavButton({ to, className, children, ...rest }: NavButtonProps) {
  const navigate = useNavigate();
  return (
    <button
      type="button"
      className={cn("cursor-pointer border-0 bg-transparent p-0 font-inherit", className)}
      onClick={() => navigate(to)}
      {...rest}
    >
      {children}
    </button>
  );
}

type RoutePressableProps = {
  readonly to: string;
  readonly className?: string;
  readonly children: ReactNode;
};

/** Block-level pressable that navigates in-app (for wrapping cards). */
export function RoutePressable({ to, className, children }: RoutePressableProps) {
  const navigate = useNavigate();
  return (
    <div
      role="link"
      tabIndex={0}
      className={cn("block cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-ring/50", className)}
      onClick={() => navigate(to)}
      onKeyDown={(e: KeyboardEvent) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          navigate(to);
        }
      }}
    >
      {children}
    </div>
  );
}
