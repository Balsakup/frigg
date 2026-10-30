import { Menu } from "@base-ui/react/menu";
import { ChevronsUpDown, LogOut, User } from "lucide-react";
import { Link, useNavigate } from "react-router";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { cn } from "~/lib/cn";
import { Icon } from "~/ui/Icon";
import { useSession } from "./useSession";

const itemClass =
  "flex w-full cursor-pointer items-center gap-3 px-4 py-2.5 text-left outline-none data-highlighted:bg-canvas";

// Menu du compte (FR-021) : identité, rôle, « Mon compte », « Se déconnecter ».
export function UserMenu({ collapsed }: { collapsed: boolean }) {
  const { account } = useSession();
  const navigate = useNavigate();
  const logout = $api.useMutation("post", "/auth/logout");
  const role = account.role === "admin" ? "Administrateur" : "Membre";

  const signOut = async () => {
    try {
      await logout.mutateAsync({});
    } finally {
      queryClient.clear();
      navigate("/connexion", { replace: true });
    }
  };

  return (
    <div className="relative shrink-0 border-t border-line p-3">
      <Menu.Root>
        <Menu.Trigger
          className={cn(
            "flex w-full items-center gap-3 rounded-md p-2 text-left hover:bg-canvas",
            collapsed && "justify-center",
          )}
        >
          <span
            aria-hidden="true"
            className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-ink/10 text-sm font-bold"
          >
            {account.email.charAt(0).toUpperCase()}
          </span>
          <span className={cn("min-w-0 flex-1 leading-tight", collapsed && "sr-only")}>
            <span className="block truncate font-bold">{account.email}</span>
            <span className="block truncate text-sm text-muted">{role}</span>
          </span>
          {collapsed ? null : <Icon icon={ChevronsUpDown} className="text-muted" />}
          <span className="sr-only">, menu du compte</span>
        </Menu.Trigger>
        <Menu.Portal>
          <Menu.Positioner side="top" align="start" sideOffset={8} className="z-50 outline-none">
            <Menu.Popup className="w-64 rounded-lg border border-line bg-surface py-1 text-ink shadow-lg outline-none">
              <Menu.LinkItem closeOnClick render={<Link to="/mon-compte" />} className={itemClass}>
                <Icon icon={User} />
                Mon compte
              </Menu.LinkItem>
              <Menu.Separator className="my-1 h-px bg-line" />
              <Menu.Item onClick={signOut} className={itemClass}>
                <Icon icon={LogOut} />
                Se déconnecter
              </Menu.Item>
            </Menu.Popup>
          </Menu.Positioner>
        </Menu.Portal>
      </Menu.Root>
    </div>
  );
}
