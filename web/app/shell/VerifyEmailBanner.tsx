import { Mail } from "lucide-react";
import { useState } from "react";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { Button } from "~/ui/Button";
import { Icon } from "~/ui/Icon";
import { useSession } from "./useSession";

// Alerte permanente tant que l'adresse n'est pas confirmée (FR-022). Pas de
// role="alert" : présente à chaque page, une annonce systématique serait du bruit.
export function VerifyEmailBanner() {
  const { account } = useSession();
  const resend = $api.useMutation("post", "/me/email-verification");
  const [status, setStatus] = useState("");
  if (account.emailVerified) return null;

  const onResend = async () => {
    setStatus("");
    try {
      await resend.mutateAsync({});
      setStatus("Un nouveau lien vient d'être envoyé.");
    } catch (error) {
      setStatus(
        isProblem(error) && error.detail ? error.detail : "L'envoi a échoué. Réessayez plus tard.",
      );
    }
  };

  return (
    <div className="mb-6 flex flex-col gap-3 rounded-lg border border-warn/40 bg-warn/5 p-4 sm:flex-row sm:items-start">
      <span className="text-warn">
        <Icon icon={Mail} />
      </span>
      <div className="min-w-0 flex-1">
        <p className="font-bold">Confirmez votre adresse électronique</p>
        <p className="mt-0.5 text-[15px]">
          Un lien de confirmation a été envoyé à{" "}
          <strong className="[overflow-wrap:anywhere]">{account.email}</strong>. Tant qu'elle n'est
          pas confirmée, vous risquez de ne pas recevoir les messages de l'association. Pensez à
          vérifier vos courriers indésirables.
        </p>
        <p role="status" className="mt-1 text-sm font-bold">
          {status}
        </p>
      </div>
      <Button
        variant="secondary"
        size="sm"
        className="shrink-0"
        onClick={onResend}
        aria-disabled={resend.isPending}
      >
        Renvoyer le lien
      </Button>
    </div>
  );
}
