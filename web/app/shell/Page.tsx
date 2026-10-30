import { type ReactNode, useEffect } from "react";
import { Breadcrumbs } from "./Breadcrumbs";
import { PageHeader } from "./PageHeader";
import { useSession } from "./useSession";

interface PageProps {
  title: string;
  intro?: string;
  actions?: ReactNode;
  breadcrumbs?: boolean;
  children?: ReactNode;
}

// Gabarit commun des pages du shell : fil d'Ariane, en-tête, contenu, et titre
// du document « Page · Association ».
export function Page({ title, intro, actions, breadcrumbs = true, children }: PageProps) {
  const { association } = useSession();
  useEffect(() => {
    document.title = `${title} · ${association.name}`;
  }, [title, association.name]);

  return (
    <>
      {breadcrumbs ? <Breadcrumbs /> : null}
      <PageHeader title={title} intro={intro} actions={actions} />
      <div className="mt-6">{children}</div>
    </>
  );
}
