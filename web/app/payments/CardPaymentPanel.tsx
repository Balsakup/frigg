import { CreditCard } from "lucide-react";
import { useId, useState } from "react";
import { isProblem } from "~/api/problem";
import { formatMoney } from "~/lib/format";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Icon } from "~/ui/Icon";

interface CardPaymentPanelProps {
  /** Lance (ou reprend) la tentative et renvoie la page HelloAsso. */
  start: () => Promise<{ redirectUrl: string }>;
  amountCents: number;
  /** Redirection vers HelloAsso ; remplaçable dans les tests. */
  redirect?: (url: string) => void;
}

// Paiement par carte via HelloAsso (008, US1) : explications, puis départ vers
// la page de paiement sécurisée de HelloAsso. Commun aux adhésions et aux
// inscriptions (012).
export function CardPaymentPanel({
  start,
  amountCents,
  redirect = (url) => window.location.assign(url),
}: CardPaymentPanelProps) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const explanation = useId();
  const amount = formatMoney(amountCents);

  const onPay = async () => {
    if (pending) return;
    setError("");
    setPending(true);
    try {
      const out = await start();
      redirect(out.redirectUrl);
    } catch (e) {
      setPending(false);
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
        <Button onClick={onPay} aria-disabled={pending} aria-describedby={explanation}>
          <Icon icon={CreditCard} />
          {pending ? "Redirection vers HelloAsso…" : `Payer ${amount} par carte`}
        </Button>
      </div>
    </div>
  );
}
