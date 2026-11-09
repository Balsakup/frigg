// Dates des événements, toujours affichées à l'heure de l'association
// (Europe/Paris), quel que soit le fuseau du navigateur (research.md 012 › R8).
const zone = "Europe/Paris";

const dayFmt = new Intl.DateTimeFormat("fr-FR", {
  timeZone: zone,
  weekday: "short",
  day: "numeric",
  month: "short",
});
const longDayFmt = new Intl.DateTimeFormat("fr-FR", {
  timeZone: zone,
  weekday: "long",
  day: "numeric",
  month: "long",
  year: "numeric",
});
const keyFmt = new Intl.DateTimeFormat("fr-CA", {
  timeZone: zone,
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
});
const partsFmt = new Intl.DateTimeFormat("fr-FR", {
  timeZone: zone,
  hour: "numeric",
  minute: "2-digit",
  hourCycle: "h23",
});
const blockFmt = new Intl.DateTimeFormat("fr-FR", {
  timeZone: zone,
  day: "numeric",
  month: "short",
});

function asDate(v: string | Date) {
  return typeof v === "string" ? new Date(v) : v;
}

/** « 14 h » ou « 14 h 30 ». */
export function formatHour(v: string | Date) {
  const parts = partsFmt.formatToParts(asDate(v));
  const h = Number(parts.find((p) => p.type === "hour")?.value);
  const m = parts.find((p) => p.type === "minute")?.value ?? "00";
  return m === "00" ? `${h} h` : `${h} h ${m}`;
}

function sameDay(a: Date, b: Date) {
  return keyFmt.format(a) === keyFmt.format(b);
}

/** « sam. 12 déc. · 14 h – 17 h » ou « du lun. 13 au ven. 17 avr. ». */
export function formatEventRange(start: string | Date, end: string | Date) {
  const s = asDate(start);
  const e = asDate(end);
  if (sameDay(s, e)) return `${dayFmt.format(s)} · ${formatHour(s)} – ${formatHour(e)}`;
  return `du ${dayFmt.format(s)} au ${dayFmt.format(e)}`;
}

/** « samedi 12 décembre 2026 à 14 h ». */
export function formatEventMoment(v: string | Date) {
  const d = asDate(v);
  // Usage français : « 1er avril », que Intl n'écrit pas.
  const day = longDayFmt.format(d).replace(/ 1 /, " 1er ");
  return `${day} à ${formatHour(d)}`;
}

/** Jour et mois abrégé pour le bloc de date des cartes : ["12", "déc."]. */
export function dateBlock(v: string | Date): [string, string] {
  const [day, month] = blockFmt.format(asDate(v)).split(" ");
  return [day ?? "", month ?? ""];
}
