import { useQuery } from "@tanstack/react-query";
import { Globe, Mail, MapPin, Phone, Share2 } from "lucide-react";
import type { ReactNode } from "react";
import { associationContactQuery } from "~/api/identity";
import { formatPhone, phoneHref } from "~/lib/format";
import { Icon } from "~/ui/Icon";

const networks = [
  ["facebook", "Facebook"],
  ["instagram", "Instagram"],
  ["linkedin", "LinkedIn"],
  ["youtube", "YouTube"],
  ["mastodon", "Mastodon"],
  ["bluesky", "Bluesky"],
] as const;

function ExternalLink({
  href,
  label,
  children,
}: {
  href: string;
  label: string;
  children: ReactNode;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      aria-label={`${label} (nouvel onglet)`}
      className="font-bold text-brand-ink underline underline-offset-2 [overflow-wrap:anywhere]"
    >
      {children}
    </a>
  );
}

function Row({ icon, children }: { icon: typeof Mail; children: ReactNode }) {
  return (
    <li className="flex items-start gap-2">
      <Icon icon={icon} className="mt-0.5 h-5 w-5 shrink-0 text-muted" />
      <span className="min-w-0">{children}</span>
    </li>
  );
}

// Encart « Contacter l'association » de l'accueil membre (010, US4, FR-025) :
// absent si aucune coordonnée ni aucun réseau n'est renseigné.
export function ContactCard({ association }: { association: string }) {
  const { data } = useQuery(associationContactQuery());
  if (!data) return null;
  const { address, contactEmail, phone, website, social } = data;
  const addressLines = [
    address.line1,
    address.line2,
    [address.postalCode, address.city].filter(Boolean).join(" "),
  ].filter((l): l is string => !!l);
  const links = networks.filter(([key]) => social[key]);
  if (!addressLines.length && !contactEmail && !phone && !website && !links.length) return null;

  return (
    <section
      aria-labelledby="contact-association"
      className="max-w-prose rounded-lg border border-line bg-surface p-5"
    >
      <h2 id="contact-association" className="font-display text-lg font-bold">
        Contacter l'association
      </h2>
      <ul className="mt-3 flex flex-col gap-2">
        {contactEmail ? (
          <Row icon={Mail}>
            <a
              href={`mailto:${contactEmail}`}
              className="font-bold text-brand-ink underline underline-offset-2 [overflow-wrap:anywhere]"
            >
              {contactEmail}
            </a>
          </Row>
        ) : null}
        {phone ? (
          <Row icon={Phone}>
            <a
              href={phoneHref(phone)}
              className="font-bold text-brand-ink underline underline-offset-2"
            >
              {formatPhone(phone)}
            </a>
          </Row>
        ) : null}
        {addressLines.length ? (
          <Row icon={MapPin}>
            <address className="not-italic">
              {addressLines.map((l) => (
                <span key={l} className="block">
                  {l}
                </span>
              ))}
            </address>
          </Row>
        ) : null}
        {website ? (
          <Row icon={Globe}>
            <ExternalLink href={website} label={`Site web de ${association}`}>
              {website.replace(/^https?:\/\//, "").replace(/\/$/, "")}
            </ExternalLink>
          </Row>
        ) : null}
        {links.map(([key, label]) => (
          <Row key={key} icon={Share2}>
            <ExternalLink href={social[key] as string} label={`${label} de ${association}`}>
              {label}
            </ExternalLink>
          </Row>
        ))}
      </ul>
    </section>
  );
}
