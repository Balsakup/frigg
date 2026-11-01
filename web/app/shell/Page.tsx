import { type ReactNode, useEffect } from "react";
import { Breadcrumbs } from "./Breadcrumbs";
import { PageHeader } from "./PageHeader";
import { useSession } from "./useSession";

interface PageProps {
  title: string;
  /** Titre du document s'il diffère du titre de la page (sous-pages). */
  documentTitle?: string;
  intro?: string;
  actions?: ReactNode;
  breadcrumbs?: boolean;
  children?: ReactNode;
}

// Gabarit commun des pages du shell : fil d'Ariane, en-tête, contenu, et titre
// du document « Page · Association ».
export function Page({
  title,
  documentTitle,
  intro,
  actions,
  breadcrumbs = true,
  children,
}: PageProps) {
  const { association } = useSession();
  useEffect(() => {
    document.title = `${documentTitle ?? title} · ${association.name}`;
  }, [documentTitle, title, association.name]);

  return (
    <>
      {breadcrumbs ? <Breadcrumbs /> : null}
      <PageHeader title={title} intro={intro} actions={actions} />
      <div className="mt-6">{children}</div>
    </>
  );
}
