import { type ReactNode, useEffect, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router";
import { readPreference, writePreference } from "~/lib/storage";
import { Alert } from "~/ui/Alert";
import { Footer } from "./Footer";
import { Header } from "./Header";
import { Sidebar } from "./Sidebar";
import { SkipLinks } from "./SkipLinks";
import { useMediaQuery } from "./useMediaQuery";
import { useSession } from "./useSession";
import { VerifyEmailBanner } from "./VerifyEmailBanner";

// Shell commun aux membres et aux administrateurs (FR-018).
export function AppShell({ children }: { children: ReactNode }) {
  const { account, association } = useSession();
  const rightsRevoked = useRightsRevoked(account.role);
  const desktop = useMediaQuery("(min-width: 1024px)");
  const [collapsed, setCollapsed] = useState(() => readPreference("sidebar-collapsed") === "1");
  const [mobileOpen, setMobileOpen] = useState(false);

  const toggleSidebar = () => {
    if (desktop) {
      setCollapsed((c) => {
        writePreference("sidebar-collapsed", c ? "0" : "1");
        return !c;
      });
    } else {
      setMobileOpen((o) => !o);
    }
  };

  const offset = desktop ? (collapsed ? "var(--sidebar-w-collapsed)" : "var(--sidebar-w)") : "0";

  return (
    <div className="font-sans text-ink">
      <SkipLinks />
      <Sidebar
        desktop={desktop}
        collapsed={collapsed}
        mobileOpen={mobileOpen}
        onMobileOpenChange={setMobileOpen}
      />
      <div
        className="flex min-h-screen flex-col transition-[padding] duration-200"
        style={{ paddingLeft: offset }}
      >
        <Header sidebarOpen={desktop ? !collapsed : mobileOpen} onToggleSidebar={toggleSidebar} />
        <main
          id="contenu"
          tabIndex={-1}
          className="flex-1 px-4 py-6 focus:outline-none lg:px-8 lg:py-8"
        >
          <VerifyEmailBanner />
          {rightsRevoked ? (
            <Alert kind="info" role="status" className="mb-6">
              Vos droits d'administration ont été retirés.
            </Alert>
          ) : null}
          {children}
        </main>
        <Footer association={association.name} />
      </div>
    </div>
  );
}

// Administrateur rétrogradé pendant sa navigation (US3) : retour à l'accueil de
// l'espace membre, avec un message tant qu'il y reste.
function useRightsRevoked(role: "member" | "admin") {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const previous = useRef(role);
  const [revoked, setRevoked] = useState(false);

  useEffect(() => {
    if (previous.current === "admin" && role === "member") {
      writePreference("last-space", "member");
      setRevoked(true);
      navigate("/", { replace: true });
    }
    previous.current = role;
  }, [role, navigate]);

  return revoked && pathname === "/";
}
