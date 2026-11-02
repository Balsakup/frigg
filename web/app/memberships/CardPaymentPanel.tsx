import { CreditCard } from "lucide-react";
import { useId, useState } from "react";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { formatMoney } from "~/lib/format";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Icon } from "~/ui/Icon";

interface CardPaymentPanelProps {
  membershipId: number;
  amountCents: number;
  /** Redirection vers HelloAsso ; remplaçable dans les tests. */
  redirect?: (url: string) => void;
}

// Paiement par carte via HelloAsso (008, US1) : explications, puis départ vers
// la page de paiement sécurisée de HelloAsso.
export function CardPaymentPanel({
  membershipId,
  amountCents,
  redirect = (url) => window.location.assign(url),
}: CardPaymentPanelProps) {
  const start = $api.useMutation("post", "/me/memberships/{id}/card-payment");
  const [error, setError] = useState("");
  const explanation = useId();
  const amount = formatMoney(amountCents);

  const onPay = async () => {
    if (start.isPending) return;
    setError("");
    try {
      const out = await start.mutateAsync({ params: { path: { id: membershipId } } });
      redirect(out.redirectUrl);
    } catch (e) {
      setError(
        isProblem(e)
          ? (e.errors?.[0]?.message ?? e.detail ?? "")
          : "Une erreur inattendue s'est produite.",
      );
    }
  };

  return (
    <div>
      <p id={explanation}>
        Vous allez être redirigé vers HelloAsso, la plateforme de paiement des associations.
        HelloAsso vous proposera une contribution volontaire facultative à son fonctionnement.
      </p>
      {error ? (
        <Alert kind="danger" role="alert" className="mt-4">
          {error}
        </Alert>
      ) : null}
      <div className="mt-4">
        <Button onClick={onPay} aria-disabled={start.isPending} aria-describedby={explanation}>
          <Icon icon={CreditCard} />
          {start.isPending ? "Redirection vers HelloAsso…" : `Payer ${amount} par carte`}
        </Button>
      </div>
    </div>
  );
}
