import {
  BookOpen,
  Calendar,
  CalendarRange,
  CircleHelp,
  ClipboardCheck,
  Coins,
  CreditCard,
  House,
  type LucideIcon,
  Mail,
  Shield,
  Users,
} from "lucide-react";

export type Space = "member" | "admin";

export interface NavEntry {
  id: string;
  label: string;
  icon: LucideIcon;
  path: string;
  space: Space;
  section: "main" | "settings";
  /** Rubrique livrée par une fonctionnalité ultérieure : page « Bientôt disponible ». */
  comingSoon?: boolean;
}

// Liste unique des rubriques (contracts/ui-routes.md › Navigation).
export const navigation: NavEntry[] = [
  { id: "accueil", label: "Accueil", icon: House, path: "/", space: "member", section: "main" },
  {
    id: "mes-membres",
    label: "Mes membres",
    icon: Users,
    path: "/mes-membres",
    space: "member",
    section: "main",
    comingSoon: true,
  },
  {
    id: "mes-inscriptions",
    label: "Mes inscriptions",
    icon: ClipboardCheck,
    path: "/mes-inscriptions",
    space: "member",
    section: "main",
    comingSoon: true,
  },
  {
    id: "a-regler",
    label: "À régler",
    icon: Coins,
    path: "/a-regler",
    space: "member",
    section: "main",
    comingSoon: true,
  },
  {
    id: "evenements",
    label: "Événements à venir",
    icon: Calendar,
    path: "/evenements",
    space: "member",
    section: "main",
    comingSoon: true,
  },
  {
    id: "adhesions",
    label: "Adhésions",
    icon: CreditCard,
    path: "/adhesions",
    space: "member",
    section: "main",
    comingSoon: true,
  },
  {
    id: "mes-documents",
    label: "Mes documents",
    icon: BookOpen,
    path: "/mes-documents",
    space: "member",
    section: "main",
    comingSoon: true,
  },
  {
    id: "aide",
    label: "Aide",
    icon: CircleHelp,
    path: "/aide",
    space: "member",
    section: "settings",
    comingSoon: true,
  },

  {
    id: "tableau-de-bord",
    label: "Tableau de bord",
    icon: House,
    path: "/admin",
    space: "admin",
    section: "main",
  },
  {
    id: "membres",
    label: "Membres",
    icon: Users,
    path: "/admin/membres",
    space: "admin",
    section: "main",
    comingSoon: true,
  },
  {
    id: "gestion-adhesions",
    label: "Adhésions",
    icon: CreditCard,
    path: "/admin/adhesions",
    space: "admin",
    section: "main",
    comingSoon: true,
  },
  {
    id: "gestion-evenements",
    label: "Événements",
    icon: Calendar,
    path: "/admin/evenements",
    space: "admin",
    section: "main",
    comingSoon: true,
  },
  {
    id: "inscriptions",
    label: "Inscriptions",
    icon: ClipboardCheck,
    path: "/admin/inscriptions",
    space: "admin",
    section: "main",
    comingSoon: true,
  },
  {
    id: "finances",
    label: "Finances",
    icon: Coins,
    path: "/admin/finances",
    space: "admin",
    section: "main",
    comingSoon: true,
  },
  {
    id: "communication",
    label: "Communication",
    icon: Mail,
    path: "/admin/communication",
    space: "admin",
    section: "main",
    comingSoon: true,
  },
  {
    id: "saisons",
    label: "Saisons",
    icon: CalendarRange,
    path: "/admin/saisons",
    space: "admin",
    section: "settings",
  },
  {
    id: "comptes",
    label: "Comptes",
    icon: Shield,
    path: "/admin/comptes",
    space: "admin",
    section: "settings",
  },
  {
    id: "aide-admin",
    label: "Aide",
    icon: CircleHelp,
    path: "/admin/aide",
    space: "admin",
    section: "settings",
    comingSoon: true,
  },
];

// L'espace se déduit de l'URL : /admin et ses sous-pages = administration.
export function spaceFromPath(pathname: string): Space {
  return pathname === "/admin" || pathname.startsWith("/admin/") ? "admin" : "member";
}

export function entriesFor(space: Space, section: NavEntry["section"]) {
  return navigation.filter((e) => e.space === space && e.section === section);
}

export function comingSoonEntries() {
  return navigation.filter((e) => e.comingSoon);
}

export function entryForPath(pathname: string) {
  return navigation.find((e) => e.path === pathname);
}
