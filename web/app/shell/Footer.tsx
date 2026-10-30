// Pied de page : liens légaux ajoutés avec leurs pages (fonctionnalité ultérieure).
export function Footer({ association }: { association: string }) {
  return (
    <footer className="border-t border-line bg-surface px-4 py-5 text-sm lg:px-8">
      <p className="text-muted">
        © {new Date().getFullYear()} {association} · Propulsé par Frigg{" "}
        <span className="tabular-nums">v{__FRIGG_VERSION__}</span>
      </p>
    </footer>
  );
}
