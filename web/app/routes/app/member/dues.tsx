import { useQuery } from "@tanstack/react-query";
import { Coins } from "lucide-react";
import { Link } from "react-router";
import { queryClient } from "~/api/query-client";
import { myDuesQuery } from "~/api/registrations";
import { formatAmount, paymentStatusLabel } from "~/lib/format";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { buttonClasses } from "~/ui/Button";
import { Icon } from "~/ui/Icon";

export const handle = { crumb: "À régler" };

export async function clientLoader() {
  await queryClient.ensureQueryData(myDuesQuery()).catch(() => undefined);
  return null;
}

// « À régler » (US4) : adhésions et inscriptions qui restent à payer.
export default function Dues() {
  const list = useQuery(myDuesQuery());
  const items = list.data?.items ?? [];
  const total = items.reduce((sum, d) => sum + d.amountCents, 0);
  return (
    <Page title="À régler" intro="Les adhésions et les inscriptions qui restent à payer.">
      {list.isError ? (
        <Alert kind="danger" role="alert" className="max-w-3xl">
          Impossible de charger les sommes à régler. Rechargez la page.
        </Alert>
      ) : list.isPending ? (
        <p className="text-muted">Chargement…</p>
      ) : items.length === 0 ? (
        <section className="max-w-3xl rounded-lg border border-line bg-surface p-6">
          <span className="grid h-11 w-11 place-items-center rounded-full bg-ok/10 text-ok">
            <Icon icon={Coins} />
          </span>
          <p className="mt-4 font-bold">Rien à régler.</p>
        </section>
      ) : (
        <div className="max-w-3xl">
          <p className="mb-4 text-lg">
            Total à régler : <span className="font-bold">{formatAmount(total)}</span>
          </p>
          <ul className="flex flex-col gap-3">
            {items.map((d) => (
              <li
                key={`${d.kind}-${d.id}`}
                className="flex flex-col gap-3 rounded-lg border border-line bg-surface p-4 sm:flex-row sm:items-center"
              >
                <div className="min-w-0 flex-1">
                  <p className="font-bold [overflow-wrap:anywhere]">{d.label}</p>
                  <p className="text-sm text-muted">
                    {d.member.firstName} {d.member.lastName} · référence {d.payment.reference}
                  </p>
                  <p className="mt-1 flex flex-wrap items-center gap-2">
                    <span className="font-bold">{formatAmount(d.amountCents)}</span>
                    <Badge tone={d.payment.status === "due" ? "warning" : "info"}>
                      {paymentStatusLabel(d.payment)}
                    </Badge>
                  </p>
                </div>
                <Link
                  to={d.kind === "membership" ? `/adhesions/${d.id}` : `/mes-inscriptions/${d.id}`}
                  className={buttonClasses()}
                >
                  Payer
                  <span className="sr-only"> {d.label}</span>
                </Link>
              </li>
            ))}
          </ul>
        </div>
      )}
    </Page>
  );
}
