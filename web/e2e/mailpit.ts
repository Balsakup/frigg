// Lecture des e-mails capturés par Mailpit (http://localhost:8025).
const MAILPIT = "http://localhost:8025/api/v1";

interface Summary {
  ID: string;
  Created: string;
}

// Attend le prochain e-mail adressé à `to` reçu après `since`, et renvoie le
// lien qu'il contient.
export async function waitForLink(to: string, since: Date, timeoutMs = 15_000): Promise<string> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const res = await fetch(`${MAILPIT}/search?query=${encodeURIComponent(`to:"${to}"`)}`);
    if (res.ok) {
      const { messages } = (await res.json()) as { messages: Summary[] };
      const latest = messages.find((m) => new Date(m.Created) >= since);
      if (latest) {
        const msg = (await (await fetch(`${MAILPIT}/message/${latest.ID}`)).json()) as {
          Text: string;
        };
        const link = msg.Text.match(/https?:\/\/\S+#[A-Za-z0-9_-]+/)?.[0];
        if (link) return link;
      }
    }
    await new Promise((r) => setTimeout(r, 300));
  }
  throw new Error(`aucun e-mail reçu pour ${to}`);
}
