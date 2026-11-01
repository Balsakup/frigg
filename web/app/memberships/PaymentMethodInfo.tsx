import type { PaymentMethodInfo as Info } from "~/api/memberships";
import { formatIban } from "~/lib/format";
import { CopyButton } from "~/ui/CopyButton";

function Row({ label, value, copy }: { label: string; value: string; copy?: string }) {
  return (
    <div className="py-2">
      <dt className="text-sm text-muted">{label}</dt>
      <dd className="flex flex-wrap items-center justify-between gap-2">
        <span className="min-w-0 font-bold [overflow-wrap:anywhere]">{value}</span>
        {copy ? (
          <CopyButton
            value={copy}
            what={label === label.toUpperCase() ? label : label.toLowerCase()}
          />
        ) : null}
      </dd>
    </div>
  );
}

// Informations d'un moyen de paiement, affichées en texte brut (jamais
// interprétées comme du HTML, research.md 006 › R7).
export function PaymentMethodInfo({ info, reference }: { info: Info; reference: string }) {
  const text = (v?: string | null) =>
    v ? <p className="mt-3 whitespace-pre-line [overflow-wrap:anywhere]">{v}</p> : null;
  switch (info.method) {
    case "cash":
      return (
        <div>
          {text(info.instructions)}
          <dl className="mt-2 divide-y divide-line">
            <Row label="Référence" value={reference} copy={reference} />
          </dl>
        </div>
      );
    case "cheque":
      return (
        <div>
          <dl className="divide-y divide-line">
            {info.payee ? <Row label="À l'ordre de" value={info.payee} /> : null}
            <Row label="Référence" value={reference} copy={reference} />
          </dl>
          {info.address ? (
            <div className="mt-2">
              <p className="text-sm text-muted">Adresse d'envoi</p>
              <p className="whitespace-pre-line font-bold">{info.address}</p>
            </div>
          ) : null}
          {text(info.instructions)}
          <p className="mt-3">Inscrivez la référence {reference} au dos du chèque.</p>
        </div>
      );
    case "transfer":
      return (
        <div>
          <dl className="divide-y divide-line">
            {info.holder ? <Row label="Titulaire du compte" value={info.holder} /> : null}
            {info.iban ? <Row label="IBAN" value={formatIban(info.iban)} copy={info.iban} /> : null}
            {info.bic ? <Row label="BIC" value={info.bic} copy={info.bic} /> : null}
            <Row label="Référence" value={reference} copy={reference} />
          </dl>
          {text(info.instructions)}
          <p className="mt-3">Indiquez la référence {reference} dans le libellé du virement.</p>
        </div>
      );
  }
}
