import type { ButtonHTMLAttributes } from "react";
import { cn } from "~/lib/cn";

export type ButtonVariant =
  | "primary"
  | "secondary"
  | "tertiary"
  | "link"
  | "danger"
  | "danger-secondary";
export type ButtonSize = "sm" | "md" | "lg";

interface ButtonStyle {
  variant?: ButtonVariant;
  size?: ButtonSize;
  block?: boolean;
  iconOnly?: boolean;
}

const variants: Record<ButtonVariant, string> = {
  primary: "bg-brand text-on-brand hover:bg-brand/90 aria-disabled:hover:bg-brand",
  secondary:
    "border border-line-strong bg-surface text-ink hover:bg-canvas aria-disabled:hover:bg-surface",
  tertiary: "text-ink hover:bg-ink/5 aria-disabled:hover:bg-transparent",
  link: "text-brand-ink underline underline-offset-2 hover:decoration-2",
  danger: "bg-danger text-white hover:bg-danger/90 aria-disabled:hover:bg-danger",
  "danger-secondary":
    "border border-danger bg-surface text-danger hover:bg-danger/5 aria-disabled:hover:bg-surface",
};

// Classes des boutons de l'artefact, réutilisables sur un lien.
export function buttonClasses({
  variant = "primary",
  size = "md",
  block,
  iconOnly,
}: ButtonStyle = {}) {
  const sizes =
    variant === "link"
      ? { sm: "text-sm", md: "", lg: "text-lg" }
      : iconOnly
        ? { sm: "h-8 w-8", md: "h-10 w-10", lg: "h-12 w-12" }
        : { sm: "h-8 gap-1.5 px-2.5 text-sm", md: "h-10 px-4", lg: "h-12 px-5 text-lg" };
  return cn(
    "inline-flex items-center justify-center gap-2 rounded-md font-bold transition-colors aria-disabled:cursor-not-allowed aria-disabled:opacity-60",
    variants[variant],
    sizes[size],
    block && "w-full",
  );
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & ButtonStyle;

// Un bouton occupé reste focalisable (aria-disabled) au lieu d'être désactivé.
export function Button({
  variant,
  size,
  block,
  iconOnly,
  className,
  type = "button",
  onClick,
  ...props
}: ButtonProps) {
  const disabled = props["aria-disabled"] === true || props["aria-disabled"] === "true";
  return (
    <button
      type={type}
      className={cn(buttonClasses({ variant, size, block, iconOnly }), className)}
      onClick={(e) => {
        if (disabled) {
          e.preventDefault();
          return;
        }
        onClick?.(e);
      }}
      {...props}
    />
  );
}
