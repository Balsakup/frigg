import { type ReactNode, useEffect, useRef } from "react";
import { useLocation } from "react-router";

interface PageHeaderProps {
  title: string;
  intro?: string;
  actions?: ReactNode;
}

// En-tête de page ; le titre reçoit le focus à chaque navigation pour que les
// lecteurs d'écran annoncent la nouvelle page.
export function PageHeader({ title, intro, actions }: PageHeaderProps) {
  const ref = useRef<HTMLHeadingElement>(null);
  const { pathname } = useLocation();
  const first = useRef(true);
  // biome-ignore lint/correctness/useExhaustiveDependencies: le focus suit le changement de page
  useEffect(() => {
    if (first.current) {
      first.current = false;
      return;
    }
    ref.current?.focus();
  }, [pathname]);

  return (
    <div className="flex flex-wrap items-end justify-between gap-x-6 gap-y-4">
      <div className="min-w-0">
        <h1
          ref={ref}
          tabIndex={-1}
          className="font-display text-[1.75rem] font-bold leading-tight focus:outline-none sm:text-3xl"
        >
          {title}
        </h1>
        {intro ? <p className="mt-1 max-w-prose text-muted">{intro}</p> : null}
      </div>
      {actions ? <div className="flex flex-wrap gap-2">{actions}</div> : null}
    </div>
  );
}
