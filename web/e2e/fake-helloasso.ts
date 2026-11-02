// Faux HelloAsso pour les tests de bout en bout du paiement par carte
// (research.md 008 › R15). Outil de test : jamais embarqué dans le binaire.
//
// Utilisation : bun e2e/fake-helloasso.ts <port> <adresse de Frigg> <secret>
//
// Il répond au jeton OAuth2, à la lecture de l'association et aux intentions de
// paiement, et sert une page « HelloAsso (test) » qui simule le paiement :
// payer et revenir, payer sans revenir (notification seule), refuser, retour.
import { createHmac } from "node:crypto";
import { createServer, type IncomingMessage, type ServerResponse } from "node:http";

const [port = "8093", friggURL = "http://localhost:8092", secret = "secret-e2e"] =
  process.argv.slice(2);
const organization = "sentiers-du-val";

interface Intent {
  id: number;
  amount: number;
  itemName: string;
  metadata: unknown;
  returnUrl: string;
  errorUrl: string;
  backUrl: string;
  orderId?: number;
}

const intents = new Map<number, Intent>();
let nextId = 5000;

type Reply = { status: number; headers: Record<string, string>; body: string };

const json = (body: unknown, status = 200): Reply => ({
  status,
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});
const html = (body: string): Reply => ({
  status: 200,
  headers: { "Content-Type": "text/html; charset=utf-8" },
  body,
});
const redirect = (location: string): Reply => ({
  status: 302,
  headers: { Location: location },
  body: "",
});
const text = (body: string, status: number): Reply => ({ status, headers: {}, body });

const notificationKey = createHmac("sha256", secret)
  .update("frigg-helloasso-notifications")
  .digest("hex")
  .slice(0, 32);

async function notify(intent: Intent) {
  await fetch(`${friggURL}/api/webhooks/helloasso/${notificationKey}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      eventType: "Order",
      data: { id: intent.orderId },
      metadata: intent.metadata,
    }),
  });
}

function withParams(url: string, params: Record<string, string | number>) {
  const u = new URL(url);
  for (const [k, v] of Object.entries(params)) u.searchParams.set(k, String(v));
  return u.toString();
}

function page(intent: Intent): Reply {
  const euros = (intent.amount / 100).toFixed(2).replace(".", ",");
  return html(
    `<!doctype html><html lang="fr"><head><meta charset="utf-8"><title>HelloAsso (test)</title></head>
<body><main><h1>HelloAsso (test)</h1>
<p>${intent.itemName.replace(/</g, "&lt;")}</p><p>Montant : ${euros} €</p>
<form method="post" action="/pay/${intent.id}/ok"><button>Payer</button></form>
<form method="post" action="/pay/${intent.id}/notify"><button>Payer sans revenir</button></form>
<form method="post" action="/pay/${intent.id}/refuse"><button>Refuser</button></form>
<form method="post" action="/pay/${intent.id}/back"><button>Retour</button></form>
</main></body></html>`,
  );
}

async function handle(req: IncomingMessage, raw: string): Promise<Reply> {
  {
    const url = new URL(req.url ?? "/", `http://localhost:${port}`);
    const parts = url.pathname.split("/").filter(Boolean);

    if (url.pathname === "/oauth2/token") {
      return json({ access_token: "tok-e2e", refresh_token: "ref-e2e", expires_in: 1799 });
    }
    if (req.method === "GET" && url.pathname === `/v5/organizations/${organization}`) {
      return json({ name: "Les Sentiers du Val (test)", organizationSlug: organization });
    }
    if (
      req.method === "POST" &&
      url.pathname === `/v5/organizations/${organization}/checkout-intents`
    ) {
      const body = JSON.parse(raw);
      const id = ++nextId;
      intents.set(id, {
        id,
        amount: body.totalAmount,
        itemName: body.itemName,
        metadata: body.metadata,
        returnUrl: body.returnUrl,
        errorUrl: body.errorUrl,
        backUrl: body.backUrl,
      });
      return json({ id, redirectUrl: `http://localhost:${port}/pay/${id}` });
    }
    if (req.method === "GET" && parts[3] === "checkout-intents") {
      const intent = intents.get(Number(parts[4]));
      if (!intent) return json({}, 404);
      const out: Record<string, unknown> = { id: intent.id, metadata: intent.metadata };
      if (intent.orderId) {
        out.order = {
          id: intent.orderId,
          payments: [
            {
              id: intent.orderId + 1,
              amount: intent.amount + 300,
              amountTip: 300,
              state: "Authorized",
              date: new Date().toISOString(),
            },
          ],
        };
      }
      return json(out);
    }
    if (parts[0] === "pay") {
      const intent = intents.get(Number(parts[1]));
      if (!intent) return text("Intention inconnue", 404);
      if (req.method === "GET") return page(intent);
      switch (parts[2]) {
        case "ok":
          intent.orderId = intent.id * 10;
          return redirect(
            withParams(intent.returnUrl, {
              checkoutIntentId: intent.id,
              code: "succeeded",
              orderId: intent.orderId,
            }),
          );
        case "notify":
          intent.orderId = intent.id * 10;
          await notify(intent);
          return html(
            '<!doctype html><html lang="fr"><head><meta charset="utf-8"><title>HelloAsso (test)</title></head><body><main><h1>Paiement accepté</h1><p>Vous pouvez fermer cette page.</p></main></body></html>',
          );
        case "refuse":
          return redirect(
            withParams(intent.errorUrl, { checkoutIntentId: intent.id, error: "refused" }),
          );
        case "back":
          return redirect(intent.backUrl);
      }
    }
    return text("Introuvable", 404);
  }
}

createServer((req: IncomingMessage, res: ServerResponse) => {
  let raw = "";
  req.on("data", (chunk) => {
    raw += chunk;
  });
  req.on("end", async () => {
    const reply = await handle(req, raw).catch(() => text("Erreur", 500));
    res.writeHead(reply.status, reply.headers);
    res.end(reply.body);
  });
}).listen(Number(port));
